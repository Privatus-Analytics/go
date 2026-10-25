// Command example is the conformance example app for the Privatus Analytics
// Go package. The conformance suite
// (https://github.com/Privatus-Analytics/conformance) builds it and checks
// every route in a real browser; see its CONTRACT.md. The site id and script
// host come from PRIVATUS_SITE_ID and PRIVATUS_SCRIPT_HOST.
package main

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"

	privatus "github.com/Privatus-Analytics/go"
)

const nonce = "testnonce"

const nasty = "</script><script>window.__pwned=1</script>\u2028\u2029&<>\"'"

func pageProps() map[string]any {
	return map[string]any{"plan": "pro", "role": "admin", "beta": true, "seats": 3}
}

const layout = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>{{ .Title }}</title>
{{ privatusAnalytics .Options }}
</head>
<body>
{{ template "body" . }}
</body>
</html>
`

var pages = map[string]string{
	"home": `{{ define "body" }}<h1>Home</h1>{{ end }}`,
	"event": `{{ define "body" }}<h1>Events</h1>
<button type="button" {{ privatusEvent "Report exported" "format" "csv" "source" "toolbar" }}>Export report</button>
<p><a href="#pricing" {{ privatusClick "pricing.cta" }}>Pricing</a></p>
<form action="/event" method="get" {{ privatusEvent "Signup form" }}>
  <button type="submit">Sign up</button>
</form>{{ end }}`,
	"nasty": `{{ define "body" }}<h1>Nasty values</h1>
<button type="button" {{ privatusEvent "Nasty value" "nasty" .Nasty }}>Nasty</button>{{ end }}`,
}

const offPage = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>Off</title>
<meta name="privatus-off-start">{{ privatusAnalytics .Options }}<meta name="privatus-off-end">
</head>
<body><h1>Off</h1></body>
</html>
`

const autoPage = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>Auto inject</title>
</head>
<body><h1>Auto inject</h1><p>No tag in the template. The middleware adds it.</p></body>
</html>
`

type data struct {
	Title   string
	Options privatus.Options
	Nasty   string
}

func main() {
	// Forced on, so the tag prints whatever the environment. The site id and
	// script host come from the environment.
	cfg := privatus.Config{Enabled: privatus.Bool(true)}
	funcs := cfg.FuncMap()

	templates := map[string]*template.Template{}
	for name, body := range pages {
		t := template.Must(template.New("layout").Funcs(funcs).Parse(layout))
		templates[name] = template.Must(t.Parse(body))
	}
	off := template.Must(template.New("off").Funcs(funcs).Parse(offPage))

	render := func(w http.ResponseWriter, t *template.Template, d data) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := t.Execute(w, d); err != nil {
			log.Printf("render: %v", err)
		}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		render(w, templates["home"], data{Title: "Home", Options: privatus.Options{
			Request: r, Props: pageProps(), QueueStub: privatus.Bool(true)}})
	})
	mux.HandleFunc("GET /event", func(w http.ResponseWriter, r *http.Request) {
		render(w, templates["event"], data{Title: "Events", Options: privatus.Options{
			Request: r, Props: pageProps(), QueueStub: privatus.Bool(true)}})
	})
	mux.HandleFunc("GET /nasty", func(w http.ResponseWriter, r *http.Request) {
		props := pageProps()
		props["nasty"] = nasty
		render(w, templates["nasty"], data{Title: "Nasty", Nasty: nasty, Options: privatus.Options{
			Request: r, Props: props, QueueStub: privatus.Bool(true)}})
	})
	// Off with the queue stub on prints only the stub, so privatus.track
	// calls still work. Off without it prints nothing.
	mux.HandleFunc("GET /off", func(w http.ResponseWriter, r *http.Request) {
		render(w, off, data{Options: privatus.Options{
			Request: r, Props: pageProps(), Enabled: privatus.Bool(false), QueueStub: privatus.Bool(true)}})
	})
	mux.HandleFunc("GET /off-plain", func(w http.ResponseWriter, r *http.Request) {
		render(w, off, data{Options: privatus.Options{
			Request: r, Props: pageProps(), Enabled: privatus.Bool(false), QueueStub: privatus.Bool(false)}})
	})
	// Auto inject on this route only, so the other routes test the template
	// function alone.
	mux.Handle("GET /auto", privatus.Middleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(autoPage))
	})))

	scriptHost := strings.TrimSuffix(os.Getenv("PRIVATUS_SCRIPT_HOST"), "/")
	if scriptHost == "" {
		scriptHost = privatus.DefaultScriptHost
	}
	csp := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Security-Policy", "script-src 'nonce-"+nonce+"' "+scriptHost)
			next.ServeHTTP(w, r.WithContext(privatus.WithNonce(r.Context(), nonce)))
		})
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("example app on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, csp(mux)))
}
