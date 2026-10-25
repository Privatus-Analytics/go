package privatus

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

const page = "<html><head><title>Hi</title></head><body>Hello</body></html>"

func TestMiddleware(t *testing.T) {
	clearEnv(t)
	htmlHandler := func(body string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("Content-Length", strconv.Itoa(len(body)))
			_, _ = io.WriteString(w, body)
		}
	}
	injected := strings.Replace(page, "</head>", basic+"\n</head>", 1)

	tests := []struct {
		name    string
		cfg     Config
		handler http.Handler
		method  string
		header  map[string]string
		want    string
		status  int
	}{
		{name: "adds the tag before the head closes", cfg: on(), handler: htmlHandler(page), want: injected},
		{name: "head close in any letter case", cfg: on(), handler: htmlHandler("<HEAD></HEAD>"), want: "<HEAD>" + basic + "\n</HEAD>"},
		{name: "sniffs a missing content type", cfg: on(), handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.WriteString(w, page)
		}), want: injected},
		{name: "explicit 200 then write", cfg: on(), handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(200)
			_, _ = io.WriteString(w, "<head>")
			_, _ = io.WriteString(w, "</head>")
		}), want: "<head>" + basic + "\n</head>"},
		{name: "no second tag", cfg: on(), handler: htmlHandler(injected), want: injected},
		{name: "leaves JSON alone", cfg: on(), handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"a":"</head>"}`)
		}), want: `{"a":"</head>"}`},
		{name: "leaves HTML without a head alone", cfg: on(), handler: htmlHandler("<p>No head here</p>"), want: "<p>No head here</p>"},
		{name: "leaves Ajax requests alone", cfg: on(), handler: htmlHandler(page), header: map[string]string{"X-Requested-With": "XMLHttpRequest"}, want: page},
		{name: "leaves encoded responses alone", cfg: on(), handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			w.Header().Set("Content-Encoding", "identity")
			_, _ = io.WriteString(w, page)
		}), want: page},
		{name: "leaves other statuses alone", cfg: on(), handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(404)
			_, _ = io.WriteString(w, page)
		}), want: page, status: 404},
		{name: "nothing outside production", cfg: Config{SiteID: "pa_TEST123456"}, handler: htmlHandler(page), want: page},
		{name: "nothing without a site id", cfg: Config{Enabled: Bool(true)}, handler: htmlHandler(page), want: page},
		{name: "empty response", cfg: on(), handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}), want: ""},
		{name: "status only", cfg: on(), handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(204)
		}), want: "", status: 204},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			method := tt.method
			if method == "" {
				method = "GET"
			}
			req := httptest.NewRequest(method, "/", nil)
			for k, v := range tt.header {
				req.Header.Set(k, v)
			}
			rec := httptest.NewRecorder()
			Middleware(tt.cfg)(tt.handler).ServeHTTP(rec, req)
			status := tt.status
			if status == 0 {
				status = 200
			}
			if rec.Code != status {
				t.Errorf("status %d, want %d", rec.Code, status)
			}
			if got := rec.Body.String(); got != tt.want {
				t.Fatalf("body\n%s\nwant\n%s", got, tt.want)
			}
			if cl := rec.Header().Get("Content-Length"); cl != "" && cl != strconv.Itoa(rec.Body.Len()) {
				t.Errorf("Content-Length %s, body is %d bytes", cl, rec.Body.Len())
			}
		})
	}
}

func TestMiddlewareFlush(t *testing.T) {
	clearEnv(t)
	tests := []struct {
		name   string
		first  string
		second string
		want   string
	}{
		{name: "head already written", first: "<head></head>", second: "<body>", want: "<head>" + basic + "\n</head><body>"},
		{name: "head not written yet streams without the tag", first: "<head>", second: "</head>", want: "<head></head>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/html")
				_, _ = io.WriteString(w, tt.first)
				w.(http.Flusher).Flush()
				_, _ = io.WriteString(w, tt.second)
			})
			rec := httptest.NewRecorder()
			Middleware(on())(h).ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
			if !rec.Flushed {
				t.Error("the underlying writer was not flushed")
			}
			if got := rec.Body.String(); got != tt.want {
				t.Fatalf("got %s, want %s", got, tt.want)
			}
		})
	}
}

func TestMiddlewareResponseController(t *testing.T) {
	clearEnv(t)
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = io.WriteString(w, "<head></head>")
		if err := http.NewResponseController(w).Flush(); err != nil {
			t.Errorf("flush: %v", err)
		}
	})
	rec := httptest.NewRecorder()
	Middleware(on())(h).ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	if !strings.Contains(rec.Body.String(), basic) || !rec.Flushed {
		t.Fatalf("got %s", rec.Body.String())
	}
}

func TestMiddlewareNonce(t *testing.T) {
	clearEnv(t)
	// A CSP middleware inside the Privatus middleware sets the nonce.
	inner := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(WithNonce(r.Context(), "inner")))
		})
	}
	h := inner(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = io.WriteString(w, page)
	}))
	rec := httptest.NewRecorder()
	Middleware(on())(h).ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	if !strings.Contains(rec.Body.String(), `nonce="inner"`) {
		t.Fatalf("inner nonce missing: %s", rec.Body.String())
	}

	// A CSP middleware outside it.
	outer := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(WithNonce(r.Context(), "outer")))
		})
	}
	rec = httptest.NewRecorder()
	outer(Middleware(on())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = io.WriteString(w, page)
	}))).ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	if !strings.Contains(rec.Body.String(), `nonce="outer"`) {
		t.Fatalf("outer nonce missing: %s", rec.Body.String())
	}
}

func TestMiddlewareRealServer(t *testing.T) {
	clearEnv(t)
	srv := httptest.NewServer(Middleware(on())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, page)
	})))
	defer srv.Close()
	res, err := http.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if !strings.Contains(string(body), basic) || res.ContentLength != int64(len(body)) {
		t.Fatalf("length %d, body %s", res.ContentLength, body)
	}
}
