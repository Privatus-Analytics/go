package privatus

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const basic = `<script defer src="https://privatusstats.com/js/pa.js" data-site="pa_TEST123456"></script>`

// clearEnv makes sure the host environment does not leak into a test.
func clearEnv(t *testing.T) {
	t.Helper()
	for _, name := range []string{"PRIVATUS_SITE_ID", "PRIVATUS_SCRIPT_HOST", "PRIVATUS_ENV", "APP_ENV", "GO_ENV"} {
		t.Setenv(name, "")
	}
}

func on() Config { return Config{SiteID: "pa_TEST123456", Enabled: Bool(true)} }

func TestTag(t *testing.T) {
	clearEnv(t)
	tests := []struct {
		name     string
		cfg      Config
		opts     Options
		want     string   // exact output, when set
		contains []string // parts the output must have
		excludes []string // parts the output must not have
	}{
		{name: "basic snippet", cfg: on(), want: basic},
		{name: "no site id", cfg: Config{Enabled: Bool(true)}, want: ""},
		{name: "not enabled outside production", cfg: Config{SiteID: "pa_TEST123456"}, want: ""},
		{name: "forced off", cfg: Config{SiteID: "pa_TEST123456", Enabled: Bool(false)}, want: ""},
		{
			name:     "modules and exclusions",
			cfg:      Config{SiteID: "pa_TEST123456", Enabled: Bool(true), Modules: []string{"engage", "auto"}, ExcludedPaths: []string{"/admin/*", "/preview/*"}},
			contains: []string{`data-modules="engage,auto"`, `data-exclude="/admin/*,/preview/*"`},
		},
		{
			name:     "an empty modules list loads none",
			cfg:      Config{SiteID: "pa_TEST123456", Enabled: Bool(true), Modules: []string{}},
			contains: []string{`data-modules=""`},
		},
		{
			name:     "domains, spa mode and manual mode",
			cfg:      Config{SiteID: "pa_TEST123456", Enabled: Bool(true), Domains: []string{"example.com, www.example.com"}, SPA: "hash", Manual: true},
			contains: []string{`data-domains="example.com,www.example.com"`, `data-spa="hash"`, `data-manual="true"`},
		},
		{name: "unknown spa mode", cfg: Config{SiteID: "pa_TEST123456", Enabled: Bool(true), SPA: "bogus"}, excludes: []string{"data-spa"}},
		{name: "auto spa mode is the default", cfg: Config{SiteID: "pa_TEST123456", Enabled: Bool(true), SPA: "auto"}, want: basic},
		{
			name:     "escapes attribute values",
			cfg:      Config{SiteID: "pa_TEST123456", Enabled: Bool(true), ExcludedPaths: []string{`/a"b`}},
			contains: []string{`data-exclude="/a&quot;b"`},
		},
		{
			name:     "per call options override the config",
			cfg:      Config{SiteID: "pa_TEST123456", Enabled: Bool(true), Modules: []string{"engage"}},
			opts:     Options{Modules: []string{"vitals"}},
			contains: []string{`data-modules="vitals"`},
		},
		{name: "per call enabled override", cfg: Config{SiteID: "pa_TEST123456"}, opts: Options{Enabled: Bool(true)}, want: basic},
		{
			name: "per call site id and host",
			cfg:  on(),
			opts: Options{SiteID: "pa_OTHER", ScriptHost: "https://stats.example.com/"},
			want: `<script defer src="https://stats.example.com/js/pa.js" data-site="pa_OTHER"></script>`,
		},
		{
			name:     "extra attributes and a nonce",
			cfg:      Config{SiteID: "pa_TEST123456", Enabled: Bool(true), ExtraAttributes: map[string]string{"data-mask": "/orders/*->/orders/:id"}},
			opts:     Options{Nonce: "abc123"},
			contains: []string{`data-mask="/orders/*-&gt;/orders/:id"`, `nonce="abc123"`},
		},
		{
			name: "invalid extra attribute names are skipped",
			cfg:  Config{SiteID: "pa_TEST123456", Enabled: Bool(true), ExtraAttributes: map[string]string{`x onload="evil"`: "1"}},
			want: basic,
		},
		{
			name: "page props",
			cfg:  on(),
			opts: Options{Props: map[string]any{"plan": "pro", "role": "admin"}},
			want: `<script>window.privatusPageProps=function(){return {"plan":"pro","role":"admin"}}</script>` + "\n" +
				`<script defer src="https://privatusstats.com/js/pa.js" data-site="pa_TEST123456" data-props="privatusPageProps"></script>`,
		},
		{
			name:     "page props with a nonce",
			cfg:      on(),
			opts:     Options{Props: map[string]any{"plan": "pro"}, Nonce: "n1"},
			contains: []string{`<script nonce="n1">window.privatusPageProps=`, `data-props="privatusPageProps" nonce="n1"></script>`},
		},
		{name: "empty props print no script", cfg: on(), opts: Options{Props: map[string]any{"plan": nil}}, want: basic},
		{
			name:     "queue stub",
			cfg:      Config{SiteID: "pa_TEST123456", Enabled: Bool(true), QueueStub: true},
			opts:     Options{Nonce: "n1"},
			contains: []string{`<script nonce="n1">` + QueueStubScript + "</script>\n<script defer"},
		},
		{
			name: "queue stub prints even when the tag is off",
			cfg:  Config{QueueStub: true},
			want: "<script>" + QueueStubScript + "</script>",
		},
		{
			name: "forced off with props and the queue stub prints only the stub",
			cfg:  Config{SiteID: "pa_TEST123456", Enabled: Bool(false), QueueStub: true},
			opts: Options{Nonce: "n1", Props: map[string]any{"plan": "pro"}},
			want: `<script nonce="n1">` + QueueStubScript + "</script>",
		},
		{
			name: "forced off with props and no queue stub prints nothing",
			cfg:  Config{SiteID: "pa_TEST123456", Enabled: Bool(false)},
			opts: Options{Props: map[string]any{"plan": "pro"}, QueueStub: Bool(false)},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := string(tt.cfg.Tag(tt.opts))
			if tt.want != "" || (tt.contains == nil && tt.excludes == nil) {
				if got != tt.want {
					t.Fatalf("got\n%s\nwant\n%s", got, tt.want)
				}
			}
			for _, part := range tt.contains {
				if !strings.Contains(got, part) {
					t.Errorf("%q missing from\n%s", part, got)
				}
			}
			for _, part := range tt.excludes {
				if strings.Contains(got, part) {
					t.Errorf("%q should not be in\n%s", part, got)
				}
			}
		})
	}
}

func TestEnvironment(t *testing.T) {
	tests := []struct {
		name                       string
		privatusEnv, appEnv, goEnv string
		want                       bool
	}{
		{name: "nothing set", want: false},
		{name: "GO_ENV production", goEnv: "production", want: true},
		{name: "APP_ENV production", appEnv: "Production", want: true},
		{name: "PRIVATUS_ENV production", privatusEnv: "production", want: true},
		{name: "development", goEnv: "development", want: false},
		{name: "first set wins", privatusEnv: "staging", appEnv: "production", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearEnv(t)
			t.Setenv("PRIVATUS_SITE_ID", "pa_FROMENV123")
			t.Setenv("PRIVATUS_ENV", tt.privatusEnv)
			t.Setenv("APP_ENV", tt.appEnv)
			t.Setenv("GO_ENV", tt.goEnv)
			if got := (Config{}).Active(); got != tt.want {
				t.Fatalf("Active() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEnvFallbacks(t *testing.T) {
	clearEnv(t)
	t.Setenv("PRIVATUS_SITE_ID", "pa_FROMENV123")
	t.Setenv("PRIVATUS_SCRIPT_HOST", "https://stats.example.com/")
	got := string(Tag(Options{Enabled: Bool(true)}))
	want := `<script defer src="https://stats.example.com/js/pa.js" data-site="pa_FROMENV123"></script>`
	if got != want {
		t.Fatalf("got %s", got)
	}
	got = string(Config{SiteID: "pa_EXPLICIT01", Enabled: Bool(true)}.Tag(Options{}))
	if !strings.Contains(got, `data-site="pa_EXPLICIT01"`) {
		t.Fatalf("explicit site id should win: %s", got)
	}
}

func TestNonceFromContextAndProps(t *testing.T) {
	clearEnv(t)
	cfg := on()
	cfg.Props = func(r *http.Request) map[string]any {
		return map[string]any{"plan": r.URL.Query().Get("plan"), "role": "admin"}
	}
	r := httptest.NewRequest("GET", "/?plan=pro", nil)
	r = r.WithContext(WithNonce(r.Context(), "ctxnonce"))
	got := string(cfg.Tag(Options{Request: r, Props: map[string]any{"role": "owner"}}))
	for _, part := range []string{`<script nonce="ctxnonce">`, `{"plan":"pro","role":"owner"}`, `nonce="ctxnonce"></script>`} {
		if !strings.Contains(got, part) {
			t.Errorf("%q missing from %s", part, got)
		}
	}
	if NonceFromContext(context.Background()) != "" {
		t.Error("empty context should have no nonce")
	}
	got = string(on().Tag(Options{Context: WithNonce(context.Background(), "c2")}))
	if !strings.Contains(got, `nonce="c2"`) {
		t.Errorf("Options.Context nonce missing: %s", got)
	}
}
