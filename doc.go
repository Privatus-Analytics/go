// Package privatus adds the cookieless Privatus Analytics tracking script to
// Go web apps.
//
// It renders the browser script tag ([Config.Tag]), HTML attributes for custom
// events and click maps ([EventAttrs], [ClickAttr]), template functions for
// html/template ([Config.FuncMap]) and an optional net/http middleware that
// puts the tag in the head of HTML pages ([Middleware]).
//
// The package only renders HTML. It never sends requests or events from your
// server, sets no cookies and never reads IP addresses, user agents or any
// visitor id.
//
// # When the tag prints
//
// The tag prints only when there is a site id (Config.SiteID or the
// PRIVATUS_SITE_ID environment variable) and the tag is enabled. Set
// Config.Enabled to force it on or off. When Enabled is nil, the tag is on
// only in production: the first non-empty one of the PRIVATUS_ENV, APP_ENV and
// GO_ENV environment variables must equal "production" (any letter case).
// When the tag is off and QueueStub is on, only the queue stub prints, so
// privatus.track calls in your own JavaScript never fail.
//
// Full docs: https://privatusanalytics.com/docs
package privatus
