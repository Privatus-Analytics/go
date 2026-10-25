package privatus

import (
	"html/template"
	"regexp"
	"strings"
)

// QueueStubScript is the official queue stub. It lets your own JavaScript call
// privatus.track before the tracker has loaded.
const QueueStubScript = "window.privatus=window.privatus||{q:[]};['track','pageview','props','optOut','optIn','ready'].forEach(function(m){window.privatus[m]=window.privatus[m]||function(){window.privatus.q.push([m,Array.prototype.slice.call(arguments)])}})"

// PagePropsFunction is the name of the global function that returns the page
// properties. The tag points to it with data-props.
const PagePropsFunction = "privatusPageProps"

var attributeName = regexp.MustCompile(`^[A-Za-z_:][A-Za-z0-9_:.-]*$`)

var htmlEscaper = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&#39;")

func escape(s string) string { return htmlEscaper.Replace(s) }

// Tag renders the script tag with a zero Config, so the site id and script
// host come from the environment. See Config.Tag.
func Tag(o Options) template.HTML { return Config{}.Tag(o) }

// Tag renders the tracker script tag, for example
//
//	<script defer src="https://privatusstats.com/js/pa.js" data-site="pa_X"></script>
//
// It renders nothing without a site id or when the tag is not enabled. With
// page properties it first renders a small inline script that defines
// window.privatusPageProps. With QueueStub on it first renders the queue stub;
// the stub prints even when the tag is off, so your own privatus.track calls
// never fail. Every attribute value is HTML-escaped.
func (c Config) Tag(o Options) template.HTML {
	s := c.resolve(o)
	var parts []string
	if s.queueStub {
		parts = append(parts, inlineScript(s.nonce, QueueStubScript))
	}
	if !s.active() {
		return template.HTML(strings.Join(parts, "\n"))
	}

	props := c.pageProps(o)
	if props != "" {
		parts = append(parts, inlineScript(s.nonce, "window."+PagePropsFunction+"=function(){return "+props+"}"))
	}

	attrs := &attrList{}
	attrs.set("defer", "", true)
	attrs.set("src", s.host+"/js/pa.js", false)
	attrs.set("data-site", s.siteID, false)
	if v := cleanList(s.modules); v != nil {
		attrs.set("data-modules", strings.Join(v, ","), false)
	}
	if v := cleanList(s.excluded); len(v) > 0 {
		attrs.set("data-exclude", strings.Join(v, ","), false)
	}
	if v := cleanList(s.domains); len(v) > 0 {
		attrs.set("data-domains", strings.Join(v, ","), false)
	}
	if spaModes[s.spa] && s.spa != "auto" {
		attrs.set("data-spa", s.spa, false)
	}
	if s.manual {
		attrs.set("data-manual", "true", false)
	}
	if props != "" {
		attrs.set("data-props", PagePropsFunction, false)
	}
	for _, name := range sortedKeys(s.extra) {
		if attributeName.MatchString(name) {
			attrs.set(name, s.extra[name], false)
		}
	}
	if s.nonce != "" {
		attrs.set("nonce", s.nonce, false)
	}
	parts = append(parts, "<script "+attrs.String()+"></script>")
	return template.HTML(strings.Join(parts, "\n"))
}

func (c Config) pageProps(o Options) string {
	merged := map[string]any{}
	if c.Props != nil && o.Request != nil {
		for k, v := range c.Props(o.Request) {
			merged[k] = v
		}
	}
	for k, v := range o.Props {
		merged[k] = v
	}
	return propsJSON(merged)
}

func inlineScript(nonce, body string) string {
	if nonce == "" {
		return "<script>" + body + "</script>"
	}
	return `<script nonce="` + escape(nonce) + `">` + body + "</script>"
}

type attr struct {
	name, value string
	bare        bool
}

type attrList struct{ items []attr }

// set adds an attribute, or replaces one with the same name in place.
func (l *attrList) set(name, value string, bare bool) {
	for i := range l.items {
		if strings.EqualFold(l.items[i].name, name) {
			l.items[i] = attr{name, value, bare}
			return
		}
	}
	l.items = append(l.items, attr{name, value, bare})
}

func (l *attrList) String() string {
	parts := make([]string, 0, len(l.items))
	for _, a := range l.items {
		if a.bare {
			parts = append(parts, a.name)
		} else {
			parts = append(parts, a.name+`="`+escape(a.value)+`"`)
		}
	}
	return strings.Join(parts, " ")
}

// cleanList keeps nil as nil. Anything else becomes trimmed, non-empty items;
// items that hold commas are split.
func cleanList(values []string) []string {
	if values == nil {
		return nil
	}
	out := []string{}
	for _, v := range values {
		for _, item := range strings.Split(v, ",") {
			if item = strings.TrimSpace(item); item != "" {
				out = append(out, item)
			}
		}
	}
	return out
}
