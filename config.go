package privatus

import (
	"context"
	"net/http"
	"os"
	"strings"
)

// Version is the version of this package.
const Version = "1.0.0"

// DefaultScriptHost serves /js/pa.js when no other host is set.
const DefaultScriptHost = "https://privatusstats.com"

// Modules lists the tracker modules you can load.
var Modules = []string{"engage", "auto", "vitals", "clicks", "ask"}

var spaModes = map[string]bool{"auto": true, "history": true, "hash": true, "off": true}

// environmentVariables are read in order. The first non-empty one wins.
var environmentVariables = []string{"PRIVATUS_ENV", "APP_ENV", "GO_ENV"}

// Config holds the settings for the tracking script. Every field is optional
// except the site id. The zero value reads the site id and script host from
// the environment.
type Config struct {
	// SiteID is the site id (it starts with pa_). Falls back to the
	// PRIVATUS_SITE_ID environment variable. Nothing prints without it.
	SiteID string
	// ScriptHost serves /js/pa.js. Falls back to PRIVATUS_SCRIPT_HOST, then
	// to DefaultScriptHost. A trailing slash is removed.
	ScriptHost string
	// Modules sets data-modules, for example []string{"engage", "auto"}.
	// nil keeps the tracker default. An empty, non-nil slice loads none.
	Modules []string
	// ExcludedPaths sets data-exclude: path globs the tracker never counts.
	ExcludedPaths []string
	// Domains sets data-domains: count only these hostnames.
	Domains []string
	// SPA is one of auto, history, hash or off. Only a value other than
	// auto is printed. Unknown values are ignored.
	SPA string
	// Manual turns off the automatic pageview (data-manual="true").
	Manual bool
	// Enabled forces the tag on or off. nil means on only in production
	// (see the package docs). Use Bool to set it.
	Enabled *bool
	// ExtraAttributes adds any other attribute, for example
	// map[string]string{"data-debug": "true"}. Names that are not valid
	// HTML attribute names are skipped.
	ExtraAttributes map[string]string
	// Props returns page properties for a request, for example the plan or
	// role. It runs only when a request is passed to Tag (Options.Request)
	// or when the Middleware renders the tag. Never return anything that
	// identifies a person.
	Props func(r *http.Request) map[string]any
	// QueueStub prints the small queue stub before the tag, so calls to
	// privatus.track work before the tracker has loaded.
	QueueStub bool
}

// Options overrides the Config for one call. Zero values keep the Config
// value. For slices, nil keeps the Config value and an empty, non-nil slice
// clears it.
type Options struct {
	SiteID        string
	ScriptHost    string
	Modules       []string
	ExcludedPaths []string
	Domains       []string
	SPA           string
	Manual        *bool
	Enabled       *bool
	QueueStub     *bool
	// ExtraAttributes are added on top of Config.ExtraAttributes.
	ExtraAttributes map[string]string
	// Props are added on top of what Config.Props returns. A key here wins.
	Props map[string]any
	// Nonce is the Content Security Policy nonce. When empty, the nonce
	// from the context (see WithNonce) is used.
	Nonce string
	// Request is the current request. It is passed to Config.Props and its
	// context is searched for a nonce.
	Request *http.Request
	// Context is searched for a nonce when Request is nil.
	Context context.Context
}

// Bool returns a pointer to v, for Config.Enabled and the Options fields.
func Bool(v bool) *bool { return &v }

// Environment returns the first non-empty value of PRIVATUS_ENV, APP_ENV
// and GO_ENV, or "" when none is set.
func Environment() string {
	for _, name := range environmentVariables {
		if v := strings.TrimSpace(os.Getenv(name)); v != "" {
			return v
		}
	}
	return ""
}

// IsProduction reports whether Environment is "production" (any letter case).
func IsProduction() bool { return strings.EqualFold(Environment(), "production") }

// Active reports whether the tag prints with this Config: there is a site id
// and the tag is enabled.
func (c Config) Active() bool { return c.resolve(Options{}).active() }

type settings struct {
	siteID, host, spa, nonce   string
	modules, excluded, domains []string
	manual, enabled, queueStub bool
	extra                      map[string]string
}

func (s settings) active() bool { return s.siteID != "" && s.enabled }

func (c Config) resolve(o Options) settings {
	s := settings{
		siteID:    firstNonEmpty(o.SiteID, c.SiteID, os.Getenv("PRIVATUS_SITE_ID")),
		host:      firstNonEmpty(o.ScriptHost, c.ScriptHost, os.Getenv("PRIVATUS_SCRIPT_HOST"), DefaultScriptHost),
		spa:       strings.TrimSpace(firstNonEmpty(o.SPA, c.SPA)),
		modules:   pick(o.Modules, c.Modules),
		excluded:  pick(o.ExcludedPaths, c.ExcludedPaths),
		domains:   pick(o.Domains, c.Domains),
		manual:    boolOr(o.Manual, c.Manual),
		queueStub: boolOr(o.QueueStub, c.QueueStub),
		nonce:     o.Nonce,
		extra:     map[string]string{},
	}
	s.host = strings.TrimSuffix(s.host, "/")
	switch {
	case o.Enabled != nil:
		s.enabled = *o.Enabled
	case c.Enabled != nil:
		s.enabled = *c.Enabled
	default:
		s.enabled = IsProduction()
	}
	if s.nonce == "" {
		s.nonce = NonceFromContext(o.context())
	}
	for k, v := range c.ExtraAttributes {
		s.extra[k] = v
	}
	for k, v := range o.ExtraAttributes {
		s.extra[k] = v
	}
	return s
}

func (o Options) context() context.Context {
	if o.Request != nil {
		return o.Request.Context()
	}
	return o.Context
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v = strings.TrimSpace(v); v != "" {
			return v
		}
	}
	return ""
}

func pick(override, base []string) []string {
	if override != nil {
		return override
	}
	return base
}

func boolOr(override *bool, base bool) bool {
	if override != nil {
		return *override
	}
	return base
}

type nonceKey struct{}

type nonceHolderKey struct{}

// WithNonce returns a copy of ctx that carries the Content Security Policy
// nonce. Tag, the template functions and the Middleware use it for every
// script they print. Call it in your CSP middleware.
func WithNonce(ctx context.Context, nonce string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	// Lets the Middleware see a nonce set by a handler or middleware that
	// runs inside it.
	if h, ok := ctx.Value(nonceHolderKey{}).(*string); ok {
		*h = nonce
	}
	return context.WithValue(ctx, nonceKey{}, nonce)
}

// NonceFromContext returns the nonce stored by WithNonce, or "".
func NonceFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if h, ok := ctx.Value(nonceHolderKey{}).(*string); ok && *h != "" {
		return *h
	}
	nonce, _ := ctx.Value(nonceKey{}).(string)
	return nonce
}
