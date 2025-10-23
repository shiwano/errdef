module github.com/shiwano/errdef/examples/zap

go 1.25.0

require (
	github.com/shiwano/errdef v0.0.0
	go.uber.org/zap v1.27.0
)

require go.uber.org/multierr v1.11.0 // indirect

replace github.com/shiwano/errdef => ../..
