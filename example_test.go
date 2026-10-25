package privatus_test

import (
	"fmt"
	"html/template"
	"net/http"
	"os"

	privatus "github.com/Privatus-Analytics/go"
)

func ExampleConfig_Tag() {
	cfg := privatus.Config{SiteID: "pa_YOURSITEID", Enabled: privatus.Bool(true)}
	fmt.Println(cfg.Tag(privatus.Options{}))
	// Output: <script defer src="https://privatusstats.com/js/pa.js" data-site="pa_YOURSITEID"></script>
}

func ExampleConfig_Tag_options() {
	cfg := privatus.Config{
		SiteID:        "pa_YOURSITEID",
		Enabled:       privatus.Bool(true),
		Modules:       []string{"engage", "auto"},
		ExcludedPaths: []string{"/admin/*"},
	}
	fmt.Println(cfg.Tag(privatus.Options{Nonce: "r4nd0m"}))
	// Output: <script defer src="https://privatusstats.com/js/pa.js" data-site="pa_YOURSITEID" data-modules="engage,auto" data-exclude="/admin/*" nonce="r4nd0m"></script>
}

func ExampleConfig_Tag_pageProps() {
	cfg := privatus.Config{SiteID: "pa_YOURSITEID", Enabled: privatus.Bool(true)}
	fmt.Println(cfg.Tag(privatus.Options{Props: map[string]any{"plan": "pro", "role": "admin"}}))
	// Output:
	// <script>window.privatusPageProps=function(){return {"plan":"pro","role":"admin"}}</script>
	// <script defer src="https://privatusstats.com/js/pa.js" data-site="pa_YOURSITEID" data-props="privatusPageProps"></script>
}

func ExampleTag() {
	// Tag uses a zero Config, so the site id comes from PRIVATUS_SITE_ID.
	os.Setenv("PRIVATUS_SITE_ID", "pa_YOURSITEID")
	defer os.Unsetenv("PRIVATUS_SITE_ID")
	fmt.Println(privatus.Tag(privatus.Options{Enabled: privatus.Bool(true)}))
	// Output: <script defer src="https://privatusstats.com/js/pa.js" data-site="pa_YOURSITEID"></script>
}

func ExampleEventAttrs() {
	fmt.Println(privatus.EventAttrs("Export CSV", map[string]any{"format": "csv", "source": "toolbar"}))
	// Output: data-privatus-event="Export CSV" data-privatus-prop-format="csv" data-privatus-prop-source="toolbar"
}

func ExampleClickAttr() {
	fmt.Println(privatus.ClickAttr("pricing.cta"))
	// Output: data-privatus-click="pricing.cta"
}

func ExampleConfig_FuncMap() {
	cfg := privatus.Config{SiteID: "pa_YOURSITEID", Enabled: privatus.Bool(true)}
	tpl := template.Must(template.New("page").Funcs(cfg.FuncMap()).Parse(
		`<head>{{ privatusAnalytics }}</head>
<button {{ privatusEvent "Report exported" "format" "csv" }}>Export</button>
<a {{ privatusClick "pricing.cta" }} href="/signup">Start</a>`))
	_ = tpl.Execute(os.Stdout, nil)
	// Output:
	// <head><script defer src="https://privatusstats.com/js/pa.js" data-site="pa_YOURSITEID"></script></head>
	// <button data-privatus-event="Report exported" data-privatus-prop-format="csv">Export</button>
	// <a data-privatus-click="pricing.cta" href="/signup">Start</a>
}

func ExampleMiddleware() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, "<html><head><title>Home</title></head><body>Hi</body></html>")
	})
	// Reads PRIVATUS_SITE_ID. Adds the tag only in production.
	handler := privatus.Middleware(privatus.Config{})(mux)
	_ = handler // http.ListenAndServe(":8080", handler)
}

func ExampleWithNonce() {
	csp := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			nonce := "r4nd0m" // make a new random value for every request
			w.Header().Set("Content-Security-Policy", "script-src 'nonce-"+nonce+"'")
			next.ServeHTTP(w, r.WithContext(privatus.WithNonce(r.Context(), nonce)))
		})
	}
	_ = csp
}
