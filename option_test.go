package errdef_test

import (
	"context"
	"errors"
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

func TestFieldOptionConstructor_WithContextFunc(t *testing.T) {
	type contextKey struct{}
	ctx := context.Background()
	ctx = context.WithValue(ctx, contextKey{}, "context_value")

	constructor, extractor := errdef.DefineField[string]("test_field")
	withContextFuncConstructor := constructor.WithContextFunc(func(ctx context.Context) string {
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

func TestFieldOptionExtractor_OrZero(t *testing.T) {
	constructor, extractor := errdef.DefineField[string]("test_field")
	orZeroExtractor := extractor.OrZero()

	option := constructor("test_value")
	err := errdef.New("test error", option)

	value := orZeroExtractor(err)
	if value != "test_value" {
		t.Errorf("want value %q, got %q", "test_value", value)
	}

	regularErr := errors.New("regular error")
	zeroValue := orZeroExtractor(regularErr)
	if zeroValue != "" {
		t.Errorf("want zero value for string, got %q", zeroValue)
	}
}

func TestFieldOptionExtractor_OrDefault(t *testing.T) {
	constructor, extractor := errdef.DefineField[string]("test_field")
	orDefaultExtractor := extractor.OrDefault("default")

	option := constructor("test_value")
	err := errdef.New("test error", option)

	value := orDefaultExtractor(err)
	if value != "test_value" {
		t.Errorf("want value %q, got %q", "test_value", value)
	}

	regularErr := errors.New("regular error")
	defaultValue := orDefaultExtractor(regularErr)
	if defaultValue != "default" {
		t.Errorf("want default value %q, got %q", "default", defaultValue)
	}
}

func TestFieldOptionExtractor_OrElse(t *testing.T) {
	constructor, extractor := errdef.DefineField[string]("test_field")
	orElseExtractor := extractor.OrElse(func(err error) string {
		return err.Error() + " default"
	})

	option := constructor("test_value")
	err := errdef.New("test error", option)

	value := orElseExtractor(err)
	if value != "test_value" {
		t.Errorf("want value %q, got %q", "test_value", value)
	}

	regularErr := errors.New("regular error")
	defaultValue := orElseExtractor(regularErr)
	if defaultValue != "regular error default" {
		t.Errorf("want default value %q, got %q", "regular error default", defaultValue)
	}
}
