# Changelog

## 1.0.0 - 2026-10-25

- Initial release.
- `Config.Tag` and `Tag` render the tracker script tag as `template.HTML`.
- `EventAttrs` and `ClickAttr` render custom event and click map attributes,
  with `EventAttributes` and `ClickAttributes` maps for templ.
- Page properties (`Config.Props`, `Options.Props`) and an optional queue stub.
- `FuncMap` with `privatusAnalytics`, `privatusEvent` and `privatusClick` for
  `html/template`.
- Optional `net/http` middleware that adds the tag to the head of HTML pages.
- Content Security Policy nonce from an option or the request context
  (`WithNonce`).
- Prints the tag only in production by default (`PRIVATUS_ENV`, `APP_ENV`,
  `GO_ENV`).
- Standard library only.
