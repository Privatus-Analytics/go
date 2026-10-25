package privatus

import (
	"bytes"
	"html/template"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFuncMap(t *testing.T) {
	clearEnv(t)
	cfg := on()
	r := httptest.NewRequest("GET", "/", nil)
	r = r.WithContext(WithNonce(r.Context(), "tplnonce"))
	tests := []struct {
		name string
		tpl  string
		data any
		want string
	}{
		{name: "tag", tpl: `<head>{{ privatusAnalytics }}</head>`, want: "<head>" + basic + "</head>"},
		{name: "tag with request nonce", tpl: `{{ privatusAnalytics . }}`, data: r, want: strings.Replace(basic, `"></script>`, `" nonce="tplnonce"></script>`, 1)},
		{name: "tag with options", tpl: `{{ privatusAnalytics . }}`, data: Options{Modules: []string{"auto"}}, want: strings.Replace(basic, `"></script>`, `" data-modules="auto"></script>`, 1)},
		{name: "event with pairs", tpl: `<button {{ privatusEvent "Export CSV" "format" "csv" }}>x</button>`,
			want: `<button data-privatus-event="Export CSV" data-privatus-prop-format="csv">x</button>`},
		{name: "event with a map", tpl: `<form {{ privatusEvent "Signup" . }}></form>`, data: map[string]any{"plan": "pro"},
			want: `<form data-privatus-event="Signup" data-privatus-prop-plan="pro"></form>`},
		{name: "event escapes", tpl: `<button {{ privatusEvent . }}>x</button>`, data: `"><script>`,
			want: `<button data-privatus-event="&quot;&gt;&lt;script&gt;">x</button>`},
		{name: "click", tpl: `<a {{ privatusClick "pricing.cta" }} href="/">x</a>`, want: `<a data-privatus-click="pricing.cta" href="/">x</a>`},
		{name: "bad click name", tpl: `<a {{ privatusClick "no good" }} href="/">x</a>`, want: `<a  href="/">x</a>`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tpl := template.Must(template.New("t").Funcs(cfg.FuncMap()).Parse(tt.tpl))
			var buf bytes.Buffer
			if err := tpl.Execute(&buf, tt.data); err != nil {
				t.Fatal(err)
			}
			if got := buf.String(); got != tt.want {
				t.Fatalf("got\n%s\nwant\n%s", got, tt.want)
			}
		})
	}
}

func TestFuncMapErrors(t *testing.T) {
	for _, src := range []string{`{{ privatusAnalytics 42 }}`, `{{ privatusEvent "E" "odd" }}`, `{{ privatusEvent "E" 1 2 }}`} {
		tpl := template.Must(template.New("t").Funcs(FuncMap()).Parse(src))
		if err := tpl.Execute(&bytes.Buffer{}, nil); err == nil {
			t.Errorf("%s: want an error", src)
		}
	}
}
