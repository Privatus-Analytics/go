package privatus

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
)

// FuncMap returns the template functions for a zero Config (site id and
// script host from the environment). See Config.FuncMap.
func FuncMap() template.FuncMap { return Config{}.FuncMap() }

// FuncMap returns template functions for html/template:
//
//	privatusAnalytics [args...]  the script tag (Config.Tag)
//	privatusEvent name [props]   event attributes (EventAttrs)
//	privatusClick name           the click map attribute (ClickAttr)
//
// privatusAnalytics takes any of: an Options value or pointer, an
// *http.Request (for the nonce and Config.Props) or a context.Context (for
// the nonce). privatusEvent takes either one map of properties or key and
// value pairs:
//
//	{{ privatusAnalytics .Request }}
//	<button {{ privatusEvent "Export CSV" "format" "csv" }}>Export</button>
//	<a {{ privatusClick "pricing.cta" }} href="/signup">Start</a>
func (c Config) FuncMap() template.FuncMap {
	return template.FuncMap{
		"privatusAnalytics": func(args ...any) (template.HTML, error) {
			var o Options
			var r *http.Request
			var ctx context.Context
			for _, arg := range args {
				switch v := arg.(type) {
				case nil:
				case Options:
					o = v
				case *Options:
					if v != nil {
						o = *v
					}
				case *http.Request:
					r = v
				case context.Context:
					ctx = v
				default:
					return "", fmt.Errorf("privatusAnalytics: unsupported argument of type %T", arg)
				}
			}
			if o.Request == nil {
				o.Request = r
			}
			if o.Context == nil {
				o.Context = ctx
			}
			return c.Tag(o), nil
		},
		"privatusEvent": func(name string, args ...any) (template.HTMLAttr, error) {
			props, err := propsFromArgs(args)
			if err != nil {
				return "", err
			}
			return EventAttrs(name, props), nil
		},
		"privatusClick": ClickAttr,
	}
}

func propsFromArgs(args []any) (map[string]any, error) {
	if len(args) == 1 {
		switch v := args[0].(type) {
		case nil:
			return nil, nil
		case map[string]any:
			return v, nil
		case map[string]string:
			m := make(map[string]any, len(v))
			for k, s := range v {
				m[k] = s
			}
			return m, nil
		}
	}
	if len(args)%2 != 0 {
		return nil, fmt.Errorf("privatusEvent: want one map or key and value pairs, got %d arguments", len(args))
	}
	m := make(map[string]any, len(args)/2)
	for i := 0; i < len(args); i += 2 {
		key, ok := args[i].(string)
		if !ok {
			return nil, fmt.Errorf("privatusEvent: property key %v is a %T, not a string", args[i], args[i])
		}
		m[key] = args[i+1]
	}
	return m, nil
}
