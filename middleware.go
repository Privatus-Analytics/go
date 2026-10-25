package privatus

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"net"
	"net/http"
	"strconv"
	"strings"
)

const marker = "/js/pa.js"

var headClose = []byte("</head>")

// Middleware returns net/http middleware that adds the tag just before
// </head> in HTML pages, so you do not have to touch your templates.
//
// It only changes responses with status 200, a text/html content type and no
// Content-Encoding header. It skips HEAD requests, Ajax requests
// (X-Requested-With: XMLHttpRequest), pages without </head> and pages that
// already load /js/pa.js. It fixes Content-Length. Other responses pass
// straight through without being held in memory.
//
// HTML pages are held in memory until the handler returns. If the handler
// calls Flush, the tag is added at that point if </head> has been written,
// and the rest of the page streams as normal. If </head> has not been
// written yet, the page streams without the tag.
//
// The nonce comes from the request context (see WithNonce). A nonce set by a
// middleware or handler running inside this one is also used.
func Middleware(cfg Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodHead ||
				strings.EqualFold(r.Header.Get("X-Requested-With"), "XMLHttpRequest") ||
				!cfg.resolve(Options{Request: r}).active() {
				next.ServeHTTP(w, r)
				return
			}
			holder := new(string)
			if n := NonceFromContext(r.Context()); n != "" {
				*holder = n
			}
			r = r.WithContext(context.WithValue(r.Context(), nonceHolderKey{}, holder))
			iw := &injectWriter{w: w, cfg: cfg, r: r}
			next.ServeHTTP(iw, r)
			iw.finish()
		})
	}
}

type writerMode int

const (
	undecided writerMode = iota
	buffering
	passthrough
)

type injectWriter struct {
	w      http.ResponseWriter
	cfg    Config
	r      *http.Request
	mode   writerMode
	status int
	buf    bytes.Buffer
}

func (iw *injectWriter) Header() http.Header { return iw.w.Header() }

// Unwrap lets http.ResponseController reach the underlying writer.
func (iw *injectWriter) Unwrap() http.ResponseWriter { return iw.w }

func (iw *injectWriter) WriteHeader(code int) {
	if iw.mode == passthrough {
		iw.w.WriteHeader(code)
		return
	}
	if iw.mode == buffering || iw.status != 0 {
		return // superfluous call, like net/http we keep the first status
	}
	if code >= 100 && code < 200 && code != http.StatusSwitchingProtocols {
		iw.w.WriteHeader(code) // informational, for example 103 Early Hints
		return
	}
	iw.status = code
	if code != http.StatusOK {
		iw.decide()
	} else if _, ok := iw.w.Header()["Content-Type"]; ok {
		iw.decide()
	}
	// A 200 without a Content-Type waits for the first Write to sniff it.
}

func (iw *injectWriter) Write(p []byte) (int, error) {
	if iw.mode == undecided {
		if iw.status == 0 {
			iw.status = http.StatusOK
		}
		h := iw.w.Header()
		if _, ok := h["Content-Type"]; !ok && iw.status == http.StatusOK && h.Get("Content-Encoding") == "" && len(p) > 0 {
			h.Set("Content-Type", http.DetectContentType(p)) // what net/http would do
		}
		iw.decide()
	}
	if iw.mode == buffering {
		return iw.buf.Write(p)
	}
	return iw.w.Write(p)
}

// Flush sends what has been written so far. See Middleware for how it
// affects the tag.
func (iw *injectWriter) Flush() {
	if iw.mode == undecided {
		if iw.status == 0 {
			iw.status = http.StatusOK
		}
		iw.decide()
	}
	if iw.mode == buffering {
		body := iw.buf.Bytes()
		injected := iw.inject(body)
		if injected != nil {
			body = injected
			iw.w.Header().Del("Content-Length") // the final length is not known yet
		}
		iw.w.WriteHeader(iw.status)
		_, _ = iw.w.Write(body)
		iw.buf.Reset()
		iw.mode = passthrough
	}
	if f, ok := iw.w.(http.Flusher); ok {
		f.Flush()
	}
}

// Hijack hands the connection to the handler, for example for WebSockets.
func (iw *injectWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := iw.w.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("privatus: the underlying ResponseWriter does not support Hijack")
	}
	iw.mode = passthrough
	return h.Hijack()
}

func (iw *injectWriter) decide() {
	h := iw.w.Header()
	html := strings.HasPrefix(strings.ToLower(strings.TrimSpace(h.Get("Content-Type"))), "text/html")
	if iw.status == http.StatusOK && html && h.Get("Content-Encoding") == "" {
		iw.mode = buffering
		return
	}
	iw.mode = passthrough
	iw.w.WriteHeader(iw.status)
}

// finish runs after the handler returns.
func (iw *injectWriter) finish() {
	switch iw.mode {
	case undecided:
		if iw.status != 0 {
			iw.w.WriteHeader(iw.status)
		}
	case buffering:
		body := iw.buf.Bytes()
		if injected := iw.inject(body); injected != nil {
			body = injected
		}
		h := iw.w.Header()
		h.Del("Content-Length")
		h.Set("Content-Length", strconv.Itoa(len(body)))
		iw.w.WriteHeader(iw.status)
		_, _ = iw.w.Write(body)
	}
}

// inject returns the body with the tag before </head>, or nil when the body
// should not change.
func (iw *injectWriter) inject(body []byte) []byte {
	if bytes.Contains(body, []byte(marker)) {
		return nil
	}
	at := indexFold(body, headClose)
	if at < 0 {
		return nil
	}
	tag := string(iw.cfg.Tag(Options{Request: iw.r}))
	if tag == "" {
		return nil
	}
	out := make([]byte, 0, len(body)+len(tag)+1)
	out = append(out, body[:at]...)
	out = append(out, tag...)
	out = append(out, '\n')
	return append(out, body[at:]...)
}

// indexFold finds an ASCII needle in s, ignoring letter case.
func indexFold(s, needle []byte) int {
	for i := 0; i+len(needle) <= len(s); i++ {
		if bytes.EqualFold(s[i:i+len(needle)], needle) {
			return i
		}
	}
	return -1
}
