package errdef_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/shiwano/errdef"
)

func TestFieldOptionConstructor_WithValue(t *testing.T) {
	constructor, extractor := errdef.DefineField[string]("test_field")
	withValueConstructor := constructor.WithValue("default_value")

	option := withValueConstructor()
	err := errdef.New("test error", option)

	value, found := extractor(err)
	if !found {
		t.Error("want field to be found")
	}
	if value != "default_value" {
		t.Errorf("want value %q, got %q", "default_value", value)
	}
}

func TestFieldOptionConstructor_WithValueFunc(t *testing.T) {
	constructor, extractor := errdef.DefineField[string]("test_field")
	withValueFuncConstructor := constructor.WithValueFunc(func() string {
		return "default_value"
	})

	option := withValueFuncConstructor()
	err := errdef.New("test error", option)

	value, found := extractor(err)
	if !found {
		t.Error("want field to be found")
	}
	if value != "default_value" {
		t.Errorf("want value %q, got %q", "default_value", value)
	}
}

func TestFieldOptionConstructor_WithContext(t *testing.T) {
	type contextKey struct{}
	ctx := context.Background()
	ctx = context.WithValue(ctx, contextKey{}, "context_value")

	constructor, extractor := errdef.DefineField[string]("test_field")
	withContextFuncConstructor := constructor.WithContext(func(ctx context.Context) string {
		return ctx.Value(contextKey{}).(string)
	})

	option := withContextFuncConstructor(ctx)
	err := errdef.New("test error", option)

	value, found := extractor(err)
	if !found {
		t.Error("want field to be found")
	}
	if value != "context_value" {
		t.Errorf("want value %q, got %q", "context_value", value)
	}
}

func TestFieldOptionConstructor_WithHTTPRequest(t *testing.T) {
	req, _ := http.NewRequest("GET", "/test/path", nil)
	req.Header.Set("X-Request-ID", "request-123")

	constructor, extractor := errdef.DefineField[string]("test_field")
	withHTTPRequestConstructor := constructor.WithHTTPRequest(func(r *http.Request) string {
		return r.Header.Get("X-Request-ID")
	})

	option := withHTTPRequestConstructor(req)
	err := errdef.New("test error", option)

	value, found := extractor(err)
	if !found {
		t.Error("want field to be found")
	}
	if value != "request-123" {
		t.Errorf("want value %q, got %q", "request-123", value)
	}
}

func TestFieldOptionExtractor_WithZero(t *testing.T) {
	constructor, extractor := errdef.DefineField[string]("test_field")
	zeroExtractor := extractor.WithZero()

	option := constructor("test_value")
	err := errdef.New("test error", option)

	value := zeroExtractor(err)
	if value != "test_value" {
		t.Errorf("want value %q, got %q", "test_value", value)
	}

	stdErr := errors.New("standard error")
	zeroValue := zeroExtractor(stdErr)
	if zeroValue != "" {
		t.Errorf("want zero value for string, got %q", zeroValue)
	}
}

func TestFieldOptionExtractor_WithDefault(t *testing.T) {
	constructor, extractor := errdef.DefineField[string]("test_field")
	defaultExtractor := extractor.WithDefault("default")

	option := constructor("test_value")
	err := errdef.New("test error", option)

	value := defaultExtractor(err)
	if value != "test_value" {
		t.Errorf("want value %q, got %q", "test_value", value)
	}

	stdErr := errors.New("standard error")
	defaultValue := defaultExtractor(stdErr)
	if defaultValue != "default" {
		t.Errorf("want default value %q, got %q", "default", defaultValue)
	}
}

func TestFieldOptionExtractor_WithFallback(t *testing.T) {
	constructor, extractor := errdef.DefineField[string]("test_field")
	fallbackExtractor := extractor.WithFallback(func(err error) string {
		return err.Error() + " fallback"
	})

	option := constructor("test_value")
	err := errdef.New("test error", option)

	value := fallbackExtractor(err)
	if value != "test_value" {
		t.Errorf("want value %q, got %q", "test_value", value)
	}

	stdErr := errors.New("standard error")
	defaultValue := fallbackExtractor(stdErr)
	if defaultValue != "standard error fallback" {
		t.Errorf("want default value %q, got %q", "standard error default", defaultValue)
	}
}

func TestFieldOptionExtractor_OrZero(t *testing.T) {
	constructor, extractor := errdef.DefineField[string]("test_field")

	option := constructor("test_value")
	err := errdef.New("test error", option)

	value := extractor.OrZero(err)
	if value != "test_value" {
		t.Errorf("want value %q, got %q", "test_value", value)
	}

	stdErr := errors.New("standard error")
	zeroValue := extractor.OrZero(stdErr)
	if zeroValue != "" {
		t.Errorf("want zero value for string, got %q", zeroValue)
	}
}

func TestFieldOptionExtractor_OrDefault(t *testing.T) {
	constructor, extractor := errdef.DefineField[string]("test_field")

	option := constructor("test_value")
	err := errdef.New("test error", option)

	value := extractor.OrDefault(err, "default")
	if value != "test_value" {
		t.Errorf("want value %q, got %q", "test_value", value)
	}

	stdErr := errors.New("standard error")
	defaultValue := extractor.OrDefault(stdErr, "default")
	if defaultValue != "default" {
		t.Errorf("want default value %q, got %q", "default", defaultValue)
	}
}

func TestFieldOptionExtractor_OrFallback(t *testing.T) {
	constructor, extractor := errdef.DefineField[string]("test_field")

	option := constructor("test_value")
	err := errdef.New("test error", option)

	value := extractor.OrFallback(err, func(err error) string {
		return err.Error() + " default"
	})
	if value != "test_value" {
		t.Errorf("want value %q, got %q", "test_value", value)
	}

	stdErr := errors.New("standard error")
	defaultValue := extractor.OrFallback(stdErr, func(err error) string {
		return err.Error() + " fallback"
	})
	if defaultValue != "standard error fallback" {
		t.Errorf("want default value %q, got %q", "standard error default", defaultValue)
	}
}
