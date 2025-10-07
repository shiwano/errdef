package errdef_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/shiwano/errdef"
)

func TestFieldConstructor_FieldKey(t *testing.T) {
	ctor1, _ := errdef.DefineField[string]("test_field_1")
	rawCtor2, _ := errdef.DefineField[int]("test_field_2")
	ctor2 := rawCtor2.WithValue(100)

	key1 := ctor1.Key()
	key2 := ctor2.Key()

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

func TestFieldConstructor_WithValue(t *testing.T) {
	ctor, extr := errdef.DefineField[string]("test_field")
	withValueCtor := ctor.WithValue("default_value")

	def := errdef.Define("test_error", withValueCtor())
	err := def.New("test error")

	got, ok := extr(err)
	if !ok {
		t.Error("want field to be found")
	}
	if want := "default_value"; got != want {
		t.Errorf("want value %q, got %q", want, got)
	}
}

func TestFieldConstructor_WithValueFunc(t *testing.T) {
	ctor, extr := errdef.DefineField[string]("test_field")
	withValueFuncCtor := ctor.WithValueFunc(func() string {
		return "default_value"
	})

	def := errdef.Define("test_error", withValueFuncCtor())
	err := def.New("test error")

	got, ok := extr(err)
	if !ok {
		t.Error("want field to be found")
	}
	if want := "default_value"; got != want {
		t.Errorf("want value %q, got %q", want, got)
	}
}

func TestFieldConstructor_WithErrorFunc(t *testing.T) {
	baseErr := errors.New("base error")

	ctor, extr := errdef.DefineField[string]("test_field")
	withErrorFuncCtor := ctor.WithErrorFunc(func(err error) string {
		return err.Error() + " processed"
	})

	def := errdef.Define("test_error")
	err := def.WithOptions(withErrorFuncCtor(baseErr)).New("test error")

	got, ok := extr(err)
	if !ok {
		t.Error("want field to be found")
	}
	if want := "base error processed"; got != want {
		t.Errorf("want value %q, got %q", want, got)
	}
}

func TestFieldConstructor_WithContextFunc(t *testing.T) {
	type contextKey struct{}
	ctx := context.Background()
	ctx = context.WithValue(ctx, contextKey{}, "context_value")

	ctor, extr := errdef.DefineField[string]("test_field")
	withContextFuncCtor := ctor.WithContextFunc(func(ctx context.Context) string {
		return ctx.Value(contextKey{}).(string)
	})

	def := errdef.Define("test_error")
	err := def.WithOptions(withContextFuncCtor(ctx)).New("test error")

	got, ok := extr(err)
	if !ok {
		t.Error("want field to be found")
	}
	if want := "context_value"; got != want {
		t.Errorf("want value %q, got %q", want, got)
	}
}

func TestFieldConstructor_WithHTTPRequestFunc(t *testing.T) {
	req, _ := http.NewRequest("GET", "/test/path", nil)
	req.Header.Set("X-Request-ID", "request-123")

	ctor, extr := errdef.DefineField[string]("test_field")
	withHTTPRequestCtor := ctor.WithHTTPRequestFunc(func(r *http.Request) string {
		return r.Header.Get("X-Request-ID")
	})

	def := errdef.Define("test_error")
	err := def.WithOptions(withHTTPRequestCtor(req)).New("test error")

	got, ok := extr(err)
	if !ok {
		t.Error("want field to be found")
	}
	if want := "request-123"; got != want {
		t.Errorf("want value %q, got %q", want, got)
	}
}

func TestFieldExtractor_WithZero(t *testing.T) {
	ctor, extr := errdef.DefineField[string]("test_field")
	zeroExtr := extr.WithZero()

	def := errdef.Define("test_error", ctor("test_value"))
	err := def.New("test error")

	got := zeroExtr(err)
	if want := "test_value"; got != want {
		t.Errorf("want value %q, got %q", want, got)
	}

	stdErr := errors.New("standard error")
	zeroValue := zeroExtr(stdErr)
	if zeroValue != "" {
		t.Errorf("want zero value for string, got %q", zeroValue)
	}
}

func TestFieldExtractor_WithDefault(t *testing.T) {
	ctor, extr := errdef.DefineField[string]("test_field")
	defaultExtr := extr.WithDefault("default")

	def := errdef.Define("test_error", ctor("test_value"))
	err := def.New("test error")

	got := defaultExtr(err)
	if want := "test_value"; got != want {
		t.Errorf("want value %q, got %q", want, got)
	}

	stdErr := errors.New("standard error")
	defaultValue := defaultExtr(stdErr)
	if want := "default"; defaultValue != want {
		t.Errorf("want default value %q, got %q", want, defaultValue)
	}
}

func TestFieldExtractor_WithFallback(t *testing.T) {
	ctor, extr := errdef.DefineField[string]("test_field")
	fallbackExtr := extr.WithFallback(func(err error) string {
		return err.Error() + " fallback"
	})

	def := errdef.Define("test_error", ctor("test_value"))
	err := def.New("test error")

	got := fallbackExtr(err)
	if want := "test_value"; got != want {
		t.Errorf("want value %q, got %q", want, got)
	}

	stdErr := errors.New("standard error")
	defaultValue := fallbackExtr(stdErr)
	if want := "standard error fallback"; defaultValue != want {
		t.Errorf("want default value %q, got %q", want, defaultValue)
	}
}

func TestFieldExtractor_OrZero(t *testing.T) {
	ctor, extr := errdef.DefineField[string]("test_field")

	def := errdef.Define("test_error", ctor("test_value"))
	err := def.New("test error")

	got := extr.OrZero(err)
	if want := "test_value"; got != want {
		t.Errorf("want value %q, got %q", want, got)
	}

	stdErr := errors.New("standard error")
	zeroValue := extr.OrZero(stdErr)
	if zeroValue != "" {
		t.Errorf("want zero value for string, got %q", zeroValue)
	}
}

func TestFieldExtractor_OrDefault(t *testing.T) {
	ctor, extr := errdef.DefineField[string]("test_field")

	def := errdef.Define("test_error", ctor("test_value"))
	err := def.New("test error")

	got := extr.OrDefault(err, "default")
	if want := "test_value"; got != want {
		t.Errorf("want value %q, got %q", want, got)
	}

	stdErr := errors.New("standard error")
	defaultValue := extr.OrDefault(stdErr, "default")
	if want := "default"; defaultValue != want {
		t.Errorf("want default value %q, got %q", want, defaultValue)
	}
}

func TestFieldExtractor_OrFallback(t *testing.T) {
	ctor, extr := errdef.DefineField[string]("test_field")

	def := errdef.Define("test_error", ctor("test_value"))
	err := def.New("test error")

	got := extr.OrFallback(err, func(err error) string {
		return err.Error() + " default"
	})
	if want := "test_value"; got != want {
		t.Errorf("want value %q, got %q", want, got)
	}

	stdErr := errors.New("standard error")
	defaultValue := extr.OrFallback(stdErr, func(err error) string {
		return err.Error() + " fallback"
	})
	if want := "standard error fallback"; defaultValue != want {
		t.Errorf("want default value %q, got %q", want, defaultValue)
	}
}
