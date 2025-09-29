package errdef_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/shiwano/errdef"
)

func TestFieldOptionConstructor_FieldKey(t *testing.T) {
	constructor1, _ := errdef.DefineField[string]("test_field_1")
	rawConstructor2, _ := errdef.DefineField[int]("test_field_2")
	constructor2 := rawConstructor2.WithValue(100)

	key1 := constructor1.FieldKey()
	key2 := constructor2.FieldKey()

	if key1.String() != "test_field_1" {
		t.Errorf("want field key %q, got %q", "test_field_1", key1.String())
	}
	if key2.String() != "test_field_2" {
		t.Errorf("want field key %q, got %q", "test_field_2", key2.String())
	}

	if key1 == key2 {
		t.Error("want different field keys for different field names")
	}
}

func TestFieldOptionConstructor_WithValue(t *testing.T) {
	constructor, extractor := errdef.DefineField[string]("test_field")
	withValueConstructor := constructor.WithValue("default_value")

	def := errdef.Define("test_error", withValueConstructor())
	err := def.New("test error")

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

	def := errdef.Define("test_error", withValueFuncConstructor())
	err := def.New("test error")

	value, found := extractor(err)
	if !found {
		t.Error("want field to be found")
	}
	if value != "default_value" {
		t.Errorf("want value %q, got %q", "default_value", value)
	}
}

func TestFieldOptionConstructor_WithContextFunc(t *testing.T) {
	type contextKey struct{}
	ctx := context.Background()
	ctx = context.WithValue(ctx, contextKey{}, "context_value")

	constructor, extractor := errdef.DefineField[string]("test_field")
	withContextFuncConstructor := constructor.WithContextFunc(func(ctx context.Context) string {
		return ctx.Value(contextKey{}).(string)
	})

	def := errdef.Define("test_error", withContextFuncConstructor(ctx))
	err := def.New("test error")

	value, found := extractor(err)
	if !found {
		t.Error("want field to be found")
	}
	if value != "context_value" {
		t.Errorf("want value %q, got %q", "context_value", value)
	}
}

func TestFieldOptionConstructor_WithHTTPRequestFunc(t *testing.T) {
	req, _ := http.NewRequest("GET", "/test/path", nil)
	req.Header.Set("X-Request-ID", "request-123")

	constructor, extractor := errdef.DefineField[string]("test_field")
	withHTTPRequestConstructor := constructor.WithHTTPRequestFunc(func(r *http.Request) string {
		return r.Header.Get("X-Request-ID")
	})

	def := errdef.Define("test_error", withHTTPRequestConstructor(req))
	err := def.New("test error")

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

	def := errdef.Define("test_error", constructor("test_value"))
	err := def.New("test error")

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

	def := errdef.Define("test_error", constructor("test_value"))
	err := def.New("test error")

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

	def := errdef.Define("test_error", constructor("test_value"))
	err := def.New("test error")

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

	def := errdef.Define("test_error", constructor("test_value"))
	err := def.New("test error")

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

	def := errdef.Define("test_error", constructor("test_value"))
	err := def.New("test error")

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

	def := errdef.Define("test_error", constructor("test_value"))
	err := def.New("test error")

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
