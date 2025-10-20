module github.com/shiwano/errdef/examples/sentry

go 1.25.0

require (
	github.com/getsentry/sentry-go v0.36.0
	github.com/shiwano/errdef v0.0.0
)

require (
	golang.org/x/sys v0.37.0 // indirect
	golang.org/x/text v0.30.0 // indirect
)

replace github.com/shiwano/errdef => ../..
