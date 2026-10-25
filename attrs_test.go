package privatus

import (
	"encoding/json"
	"strings"
	"testing"
)

type plan string

func (p plan) String() string { return "plan:" + string(p) }

func TestEventAttrs(t *testing.T) {
	tests := []struct {
		name  string
		event string
		props map[string]any
		want  string
	}{
		{
			name:  "spec example",
			event: "Export CSV",
			props: map[string]any{"format": "csv", "source": "toolbar"},
			want:  `data-privatus-event="Export CSV" data-privatus-prop-format="csv" data-privatus-prop-source="toolbar"`,
		},
		{name: "no props", event: "Signup", want: `data-privatus-event="Signup"`},
		{name: "empty name", event: "   ", props: map[string]any{"a": "b"}, want: ""},
		{name: "name is trimmed", event: "  Signup  ", want: `data-privatus-event="Signup"`},
		{name: "name is cut to 120 characters", event: strings.Repeat("é", 130), want: `data-privatus-event="` + strings.Repeat("é", 120) + `"`},
		{
			name:  "keys are normalised",
			event: "E",
			props: map[string]any{"Report Type!": "a", "plan-tier": "b"},
			want:  `data-privatus-event="E" data-privatus-prop-plan_tier="b" data-privatus-prop-report_type_="a"`,
		},
		{
			name:  "keys are cut to 64 characters",
			event: "E",
			props: map[string]any{strings.Repeat("k", 70): "v"},
			want:  `data-privatus-event="E" data-privatus-prop-` + strings.Repeat("k", 64) + `="v"`,
		},
		{
			name:  "value types become text",
			event: "E",
			props: map[string]any{"b": true, "f": 1.5, "i": 42, "l": []string{"x", "y"}, "n": nil, "s": plan("pro"), "u": uint8(7)},
			want:  `data-privatus-event="E" data-privatus-prop-b="true" data-privatus-prop-f="1.5" data-privatus-prop-i="42" data-privatus-prop-l="x,y" data-privatus-prop-s="plan:pro" data-privatus-prop-u="7"`,
		},
		{
			name:  "values are escaped",
			event: `Say "hi" <now>`,
			props: map[string]any{"q": `a&b'c`},
			want:  `data-privatus-event="Say &quot;hi&quot; &lt;now&gt;" data-privatus-prop-q="a&amp;b&#39;c"`,
		},
		{
			name:  "values are cut to 500 characters",
			event: "E",
			props: map[string]any{"v": strings.Repeat("x", 600)},
			want:  `data-privatus-event="E" data-privatus-prop-v="` + strings.Repeat("x", 500) + `"`,
		},
		{
			name:  "nil pointers are dropped",
			event: "E",
			props: map[string]any{"p": (*string)(nil)},
			want:  `data-privatus-event="E"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := string(EventAttrs(tt.event, tt.props)); got != tt.want {
				t.Fatalf("got\n%s\nwant\n%s", got, tt.want)
			}
		})
	}
}

func TestEventAttrsKeepsThirtyProps(t *testing.T) {
	props := map[string]any{}
	for i := 0; i < 40; i++ {
		props[string(rune('a'+i/26))+string(rune('a'+i%26))] = i
	}
	if got := strings.Count(string(EventAttrs("E", props)), "data-privatus-prop-"); got != 30 {
		t.Fatalf("got %d props, want 30", got)
	}
}

func TestEventAttributes(t *testing.T) {
	got := EventAttributes("Export CSV", map[string]any{"Format": "csv"})
	if len(got) != 2 || got["data-privatus-event"] != "Export CSV" || got["data-privatus-prop-format"] != "csv" {
		t.Fatalf("got %v", got)
	}
	if len(EventAttributes("", nil)) != 0 {
		t.Fatal("empty name should give an empty map")
	}
}

func TestClickAttr(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"pricing.cta", `data-privatus-click="pricing.cta"`},
		{"nav:signup-button_2", `data-privatus-click="nav:signup-button_2"`},
		{"", ""},
		{"has space", ""},
		{`quote"`, ""},
		{strings.Repeat("a", 100), `data-privatus-click="` + strings.Repeat("a", 100) + `"`},
		{strings.Repeat("a", 101), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := string(ClickAttr(tt.name)); got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
			if m := ClickAttributes(tt.name); (len(m) == 1) != (tt.want != "") {
				t.Fatalf("ClickAttributes(%q) = %v", tt.name, m)
			}
		})
	}
}

func TestPropsJSON(t *testing.T) {
	long := make([]string, 40)
	for i := range long {
		long[i] = "x"
	}
	tests := []struct {
		name  string
		props map[string]any
		want  string
	}{
		{name: "empty", props: nil, want: ""},
		{name: "types are kept", props: map[string]any{"pro": true, "seats": 5, "ratio": 0.25, "flags": []string{"a", "b"}},
			want: `{"flags":["a","b"],"pro":true,"ratio":0.25,"seats":5}`},
		{name: "safe inside a script", props: map[string]any{"x": "</script><b>&\u2028\u2029"},
			want: `{"x":"\u003c/script\u003e\u003cb\u003e\u0026\u2028\u2029"}`},
		{name: "keys are normalised", props: map[string]any{"App Version": "1.2"}, want: `{"app_version":"1.2"}`},
		{name: "nil is dropped", props: map[string]any{"a": nil, "b": "c"}, want: `{"b":"c"}`},
		{name: "json numbers stay numbers", props: map[string]any{"n": json.Number("12")}, want: `{"n":12}`},
		{name: "lists keep 30 items", props: map[string]any{"l": long}, want: `{"l":[` + strings.TrimSuffix(strings.Repeat(`"x",`, 30), ",") + `]}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := propsJSON(tt.props); got != tt.want {
				t.Fatalf("got  %s\nwant %s", got, tt.want)
			}
		})
	}
}
