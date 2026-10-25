module github.com/Privatus-Analytics/go/example

go 1.22

require github.com/Privatus-Analytics/go v1.0.0

// The package from this repo's own source, not from the module proxy.
replace github.com/Privatus-Analytics/go => ../
