package privatus

import (
	"html/template"
	"regexp"
	"strings"
)

var clickName = regexp.MustCompile(`^[A-Za-z0-9_.:-]{1,100}$`)

// EventAttrs renders the attributes that send a custom event when someone
// clicks the element, or submits it when the element is a <form>:
//
//	EventAttrs("Export CSV", map[string]any{"format": "csv"})
//	// data-privatus-event="Export CSV" data-privatus-prop-format="csv"
//
// The name is trimmed and cut to 120 characters; an empty name renders
// nothing. Property keys are lowercased, any character other than a-z, 0-9
// and _ becomes _, and keys are cut to 64 characters. At most 30 properties
// are kept, in key order. Values become text (true/false, numbers as digits,
// lists joined with commas), nil values are dropped and values are cut to
// 500 characters. Everything is HTML-escaped.
//
// It renders even when the tag is off, so your HTML is the same in
// development and production.
func EventAttrs(name string, props map[string]any) template.HTMLAttr {
	name = cut(strings.TrimSpace(name), maxEventName)
	if name == "" {
		return ""
	}
	var b strings.Builder
	b.WriteString(`data-privatus-event="` + escape(name) + `"`)
	for _, p := range normalizeProps(props, false) {
		b.WriteString(` data-privatus-prop-` + p.key + `="` + escape(p.value.(string)) + `"`)
	}
	return template.HTMLAttr(b.String())
}

// EventAttributes returns the same attributes as EventAttrs as a map, for
// template engines that escape attribute values themselves, such as templ:
//
//	<button { templ.Attributes(privatus.EventAttributes("Export CSV", nil))... }>
//
// The map is empty when the name is empty.
func EventAttributes(name string, props map[string]any) map[string]any {
	out := map[string]any{}
	name = cut(strings.TrimSpace(name), maxEventName)
	if name == "" {
		return out
	}
	out["data-privatus-event"] = name
	for _, p := range normalizeProps(props, false) {
		out["data-privatus-prop-"+p.key] = p.value
	}
	return out
}

// ClickAttr names an element for click maps:
//
//	ClickAttr("pricing.cta") // data-privatus-click="pricing.cta"
//
// The name must match ^[A-Za-z0-9_.:-]{1,100}$. Any other name renders
// nothing.
func ClickAttr(name string) template.HTMLAttr {
	if !clickName.MatchString(name) {
		return ""
	}
	return template.HTMLAttr(`data-privatus-click="` + name + `"`)
}

// ClickAttributes returns the same attribute as ClickAttr as a map, for
// templ and similar engines. The map is empty when the name is not valid.
func ClickAttributes(name string) map[string]any {
	if !clickName.MatchString(name) {
		return map[string]any{}
	}
	return map[string]any{"data-privatus-click": name}
}
