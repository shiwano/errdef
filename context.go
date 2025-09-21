package errdef

import "context"

type contextKey struct{}

var optionsFromContextKey = contextKey{}

// ContextWithOptions adds error options to a context.
// These options will be automatically applied when creating errors
// using Definition.With method.
func ContextWithOptions(ctx context.Context, opts ...Option) context.Context {
	if len(opts) == 0 {
		return ctx
	}
	ctxOpts := optionsFromContext(ctx)
	newOpts := make([]Option, len(ctxOpts)+len(opts))
	copy(newOpts, ctxOpts)
	copy(newOpts[len(ctxOpts):], opts)
	return context.WithValue(ctx, optionsFromContextKey, newOpts)
}

func optionsFromContext(ctx context.Context) []Option {
	if ctx == nil {
		return nil
	}
	rawOts := ctx.Value(optionsFromContextKey)
	if rawOts == nil {
		return nil
	}
	opts, ok := rawOts.([]Option)
	if !ok {
		return nil
	}
	return opts
}
