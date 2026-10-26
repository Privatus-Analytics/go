# Privatus Analytics for Go

[![Go Reference](https://pkg.go.dev/badge/github.com/Privatus-Analytics/go.svg)](https://pkg.go.dev/github.com/Privatus-Analytics/go)

**Real visitor numbers for your Go web app. No cookies, no cookie pop-up, no bots.**

You want to know if people use your Go web app. But Google Analytics uses
cookies, so in many countries you must show a cookie pop-up first (the box that
asks visitors to say yes to cookies). Many people click no, so they never show
up in your numbers. Bots add fake visits on top.

This package adds the Privatus Analytics script to your Go web app with
html/template functions, or a net/http middleware that puts it in the page head
for you. It uses only the Go standard library. The script sets no cookies and
keeps no IP addresses. Bots are filtered out before they are counted.

So you see how many real people came, which pages they looked at and where they
came from, on one simple dashboard. A report comes to your inbox each week, and
you can set alerts for when traffic jumps or drops. And you don't need a cookie
pop-up for your analytics.

**Free plan, no card:** 10,000 events a month (an event is a page view or an action you track) on up to 3 sites. [Start free at privatusanalytics.com](https://privatusanalytics.com).

- **No cookies.** No cookies and no hidden IDs on your visitors' devices, and no IP
  addresses are kept. Laws differ by country, so check your own legal advice.
- **Real people, not bots.** Known bots, data centers and spam visits are left
  out on every plan.
- **Small and fast.** The script is about 3 KB when compressed.

## What this package does

- `html/template` functions, or an optional `net/http` middleware that puts the
  script in the `<head>` for you.
- The package only prints HTML for the browser. It never sends events or any
  other request from your server, sets no cookies, and never reads IP
  addresses, browser details (user agents) or any visitor id.
- It uses only the Go standard library.

You need a Privatus Analytics account to see your reports, and there is a
free plan. [Sign up at privatusanalytics.com](https://privatusanalytics.com).

## Requirements

* Go 1.22 or later.
* Works with `net/http`, `html/template`, templ, Gin, Echo and anything else
  that speaks `net/http`.
* A Privatus Analytics site id (it starts with `pa_`). You find it under
  Site settings, Tracking.

## Install

```sh
go get github.com/Privatus-Analytics/go
```

The package name is `privatus`:

```go
import privatus "github.com/Privatus-Analytics/go"
```

For the full setup guide and every tracker option, see the [Privatus
Analytics docs](https://privatusanalytics.com/docs).

Set your site id with an environment variable:

```sh
PRIVATUS_SITE_ID=pa_YOURSITEID
```

or in code:

```go
var analytics = privatus.Config{SiteID: "pa_YOURSITEID"}
```

## Add the tag

Add the template functions, then put the tag in the `<head>` of your layout:

```go
tpl := template.Must(template.New("layout").
	Funcs(analytics.FuncMap()).
	ParseFiles("layout.html"))
```

```html
<head>
  <title>My app</title>
  {{ privatusAnalytics }}
</head>
```

In production it prints:

```html
<script defer src="https://privatusstats.com/js/pa.js" data-site="pa_YOURSITEID"></script>
```

The tag prints nothing when there is no site id, and nothing outside
production (see below). The one exception is the queue stub: when you turn it
on, it still prints, alone.

You can also call it from Go code. It returns `template.HTML`:

```go
html := analytics.Tag(privatus.Options{})
html = privatus.Tag(privatus.Options{}) // uses only the environment variables
```

### Content Security Policy nonce

A Content Security Policy (CSP) can block scripts that do not have the right
one-time code, called a nonce. Store the nonce in the request context with
`WithNonce`, and every script this package prints gets it:

```go
func csp(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nonce := newRandomNonce() // a fresh random value for every request
		w.Header().Set("Content-Security-Policy", "script-src 'nonce-"+nonce+"'")
		next.ServeHTTP(w, r.WithContext(privatus.WithNonce(r.Context(), nonce)))
	})
}
```

In templates, pass the request so the tag can find the nonce:

```html
{{ privatusAnalytics .Request }}
```

Or pass it yourself: `analytics.Tag(privatus.Options{Nonce: nonce})`.

### Or let the package add it

If you would rather not touch your templates, wrap your handler with the
middleware:

```go
mux := http.NewServeMux()
// ... your routes
http.ListenAndServe(":8080", privatus.Middleware(analytics)(mux))
```

It adds the tag just before `</head>` in uncompressed `text/html` responses with
status 200, and fixes the `Content-Length` header. It skips `HEAD` requests,
Ajax requests (the `X-Requested-With: XMLHttpRequest` header), pages without a
`<head>`, and pages that already have the tag. Responses that have a
`Content-Encoding` header are left alone, so put the middleware inside (after)
any compression middleware, or use the template function instead.

Other responses, such as JSON, files and errors, pass straight through and are
never held in memory. HTML pages are held in memory until your handler
returns. If your handler streams a page with `Flush`, the tag is added at the
first flush when `</head>` has already been written. If it has not, that page
streams without the tag.

The nonce from `WithNonce` works whether your CSP middleware runs before or
after this one.

## Configuration

Every setting is optional except the site id.

```go
var analytics = privatus.Config{
	SiteID:          "pa_YOURSITEID",             // or PRIVATUS_SITE_ID
	ScriptHost:      "https://privatusstats.com", // or PRIVATUS_SCRIPT_HOST
	Modules:         []string{"engage", "auto", "vitals", "clicks", "ask"},
	ExcludedPaths:   []string{"/admin/*", "/preview/*"},
	Domains:         []string{"example.com", "www.example.com"},
	SPA:             "auto", // auto, history, hash or off
	Manual:          false,  // true turns off the automatic pageview
	Enabled:         nil,    // privatus.Bool(true) or privatus.Bool(false) forces it
	ExtraAttributes: map[string]string{"data-mask": "/orders/*->/orders/:id"},
	QueueStub:       false,
	Props: func(r *http.Request) map[string]any {
		return map[string]any{"plan": planFor(r)}
	},
}
```

| Setting | Default | What it does |
|---|---|---|
| `SiteID` | `PRIVATUS_SITE_ID` | The `data-site` attribute. Nothing prints without it |
| `ScriptHost` | `https://privatusstats.com` | Where `/js/pa.js` is served (or `PRIVATUS_SCRIPT_HOST`). Change it only if you host your own copy |
| `Modules` | tracker default (`engage`) | The `data-modules` attribute. Setting it replaces the default, so keep `engage` for time on page and scroll depth. `[]string{}` loads none |
| `ExcludedPaths` | none | The `data-exclude` attribute. Path patterns the tracker never counts |
| `Domains` | all hosts | The `data-domains` attribute. Count only these hostnames |
| `SPA` | `auto` | The `data-spa` attribute, for single-page apps |
| `Manual` | `false` | The `data-manual` attribute |
| `Enabled` | `nil` (production only) | `privatus.Bool(true)` or `privatus.Bool(false)` forces the tag on or off everywhere |
| `ExtraAttributes` | none | Any other `data-*` attribute |
| `Props` | none | Page properties for each request (see below) |
| `QueueStub` | `false` | Prints the queue stub before the tag (see below) |
| auto inject | off | Wrap your handler with `privatus.Middleware(cfg)` |

Each tracker attribute is explained in the [script attributes
reference](https://privatusanalytics.com/docs/tracker/attributes), and the
modules in [Modules](https://privatusanalytics.com/docs/tracker/modules).

You can also override any setting for one page with `Options`:

```html
{{ privatusAnalytics .TagOptions }}
```

```go
data.TagOptions = privatus.Options{
	Modules:       []string{"engage", "auto"},
	ExcludedPaths: []string{"/preview/*"},
	Request:       r,
}
```

`privatusAnalytics` takes any of: an `Options` value, an `*http.Request` or a
`context.Context`. For list settings in `Options`, `nil` keeps the `Config`
value and an empty list clears it.

### Development and test

When `Enabled` is `nil`, the tag prints only in production. The package looks
at the first of these environment variables that is set: `PRIVATUS_ENV`, then
`APP_ENV`, then `GO_ENV`. The tag prints when that value is `production` (any
letter case). So `PRIVATUS_ENV=staging` turns the tag off even when
`APP_ENV=production`.

Development, test and staging requests are not counted by default, so your own
clicking around never reaches your reports. To see the tag while you build, set
`Enabled: privatus.Bool(true)`. The tracker also ignores `localhost` unless you
pass `data-allow-local` or `data-debug` (see [Debug
mode](https://privatusanalytics.com/docs/tracker/debug)):

```go
analytics.ExtraAttributes = map[string]string{"data-debug": "true"}
```

### Leaving people out

Use `data-exclude` through `ExcludedPaths` for admin areas. To leave out
signed-in staff, print the tag only for other people:

```html
{{ if not .User.IsAdmin }}{{ privatusAnalytics .Request }}{{ end }}
```

## Product analytics: see which features people use

### Buttons and forms

`privatusEvent` (or `EventAttrs` in Go code) adds the attributes that send a
custom event when someone clicks the element:

```html
<button {{ privatusEvent "Report exported" "format" "csv" "source" "toolbar" }}>Export CSV</button>
```

prints:

```html
<button data-privatus-event="Report exported" data-privatus-prop-format="csv" data-privatus-prop-source="toolbar">Export CSV</button>
```

Pass the properties as key and value pairs, or as one `map[string]any`. On a
`<form>`, the event is sent when the form is submitted:

```html
<form method="post" action="/signup" {{ privatusEvent "Signup completed" "plan" "pro" }}>
```

Some rules, so the data stays clean:

* The event name is trimmed and cut to 120 characters. An empty name prints
  nothing.
* Property names are lowercased, and any character other than `a-z`, `0-9`
  and `_` becomes `_`. They are cut to 64 characters. At most 30 properties
  are kept.
* Values become text: `true` or `false`, numbers as digits, lists joined
  with commas. Empty (`nil`) values are dropped. Values are cut to 500
  characters.
* Everything is escaped for HTML.
* The attributes print even when the tag is off, so your HTML is the same in
  development and production.

### Click maps

Give important links and buttons a stable name for [click
maps](https://privatusanalytics.com/docs/features/click-maps):

```html
<a {{ privatusClick "pricing.cta" }} href="/signup">Start free</a>
```

prints `data-privatus-click="pricing.cta"`. The name may use letters, digits
and `_ . : -`, up to 100 characters. Any other name prints nothing.

### Page properties

Page properties are sent with every pageview and event on the page. Use them
to split your reports by plan, role, app version, feature flag or account
tier. Set them for every request with `Config.Props`, or for one page with
`Options.Props` (a key there wins):

```go
analytics.Props = func(r *http.Request) map[string]any {
	return map[string]any{"plan": "pro", "role": "admin"}
}
```

`Config.Props` runs when you pass the request (`{{ privatusAnalytics .Request }}`
or `Options{Request: r}`), and when the middleware adds the tag. The
middleware passes the request as it saw it, so values that a later middleware
puts in the context are not there yet. In that case print the tag from your
template.

The tag then prints a small script first, and points to it with `data-props`:

```html
<script>window.privatusPageProps=function(){return {"plan":"pro","role":"admin"}}</script>
<script defer src="https://privatusstats.com/js/pa.js" data-site="pa_YOURSITEID" data-props="privatusPageProps"></script>
```

Names follow the same rules as event properties. Numbers and booleans stay
numbers and booleans, and lists stay lists (up to 30 items). The script is
safe to print in the page: characters that could end it early are escaped.

**Never put names, email addresses, user ids or anything else that points to
one person in page properties.** Privatus Analytics counts visitors without any id, so
put groups (like plan or role), not people. The server also removes email
addresses and long ids it finds.

### Track from your own JavaScript

Call `privatus.track` with an event name and its properties:

```js
privatus.track("Feature used", { feature: "export" });
```

If your code can run before the tracker has loaded, turn on the queue stub.
It is a tiny script that keeps your calls in a queue until the tracker is
ready:

```go
analytics.QueueStub = true
```

It prints this before the tag (with your nonce, if you have one), even when
the tag is off, so your calls never fail in development:

```html
<script>window.privatus=window.privatus||{q:[]};['track','pageview','props','optOut','optIn','ready'].forEach(function(m){window.privatus[m]=window.privatus[m]||function(){window.privatus.q.push([m,Array.prototype.slice.call(arguments)])}})</script>
```

### Naming events

Use the "Noun verb" style, in the past tense: `Report exported`, `Invite sent`,
`Plan upgraded`. Keep the name short and the same everywhere, and put the
details in properties (`format: csv`), not in the name. Then you get one event
to count, with a breakdown by property.

Next steps:

* Build [funnels](https://privatusanalytics.com/docs/features/funnels) from
  your events to see where people stop.
* Turn events into [goals](https://privatusanalytics.com/docs/features/goals)
  to track conversions.
* See every event and property in the [events
  docs](https://privatusanalytics.com/docs/events).

## templ, Gin and Echo

The package does not depend on any of these. These are examples only.

### templ

[templ](https://templ.guide) escapes values itself, so use `templ.Raw` for the
tag, and the map helpers `EventAttributes` and `ClickAttributes` for
attributes. templ has its own nonce in the context:

```templ
templ Layout(title string) {
	<html>
		<head>
			<title>{ title }</title>
			@templ.Raw(string(analytics.Tag(privatus.Options{Context: ctx, Nonce: templ.GetNonce(ctx)})))
		</head>
		<body>
			<button { templ.Attributes(privatus.EventAttributes("Report exported", map[string]any{"format": "csv"}))... }>Export</button>
			<a { templ.Attributes(privatus.ClickAttributes("pricing.cta"))... } href="/signup">Start free</a>
			{ children... }
		</body>
	</html>
}
```

### Gin

```go
r := gin.Default()
r.SetFuncMap(analytics.FuncMap())
r.LoadHTMLGlob("templates/*")
r.GET("/", func(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", gin.H{"Request": c.Request})
})

// Or add the tag to every page: a Gin engine is an http.Handler.
http.ListenAndServe(":8080", privatus.Middleware(analytics)(r))
```

### Echo

```go
e := echo.New()
e.Renderer = &renderer{tpl: template.Must(template.New("").
	Funcs(analytics.FuncMap()).ParseGlob("templates/*.html"))}

// Or add the tag to every page:
e.Use(echo.WrapMiddleware(privatus.Middleware(analytics)))
```

where `renderer` is the usual small `echo.Renderer` that calls
`tpl.ExecuteTemplate`.

## Privacy

The script sets no cookies and stores nothing in the browser. This package
never sends anything from your server. See the [privacy
documentation](https://privatusanalytics.com/docs/privacy) for what is
collected.

If you have a privacy policy, it should say which tools handle your visitors' data. You can paste this:

> This website uses [Privatus Analytics](https://privatusanalytics.com/privacy-analytics) to count visits. It
> does not use cookies and does not follow you from one website to another. Your IP address is used for a moment
> to count visits and is never stored.

Change it to fit your site. The [privacy policy generator](https://privatusanalytics.com/tools/privacy-policy-generator)
makes a longer version with your site's name in it.

## Development

The checks run in Docker so your local Go does not matter:

```sh
./run-docker.sh
```

It runs `gofmt`, `go vet`, `staticcheck`, the tests (with the race detector)
and `go build` in the official `golang:1.23` image.

## Contributing

Bug reports and pull requests are welcome at
https://github.com/Privatus-Analytics/go.

## License

MIT. See [LICENSE](LICENSE). Privatus Analytics is a product of Two Phase LLC,
Wyoming, USA.
