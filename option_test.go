package errdef_test

import (
	"context"
	"errors"
	"testing"

	"github.com/shiwano/errdef"
)

func TestFieldOptionConstructor_Default(t *testing.T) {
	constructor, extractor := errdef.DefineField[string]("test_field")
	defaultConstructor := constructor.Default("default_value")

	option := defaultConstructor()
	err := errdef.New("test error", option)

	value, found := extractor(err)
	if !found {
		t.Error("want field to be found")
	}
	if value != "default_value" {
		t.Errorf("want value %q, got %q", "default_value", value)
	}
}

func TestFieldOptionConstructor_DefaultFunc(t *testing.T) {
	constructor, extractor := errdef.DefineField[string]("test_field")
	defaultFuncConstructor := constructor.DefaultFunc(func() string {
		return "default_value"
	})

	option := defaultFuncConstructor()
	err := errdef.New("test error", option)

	value, found := extractor(err)
	if !found {
		t.Error("want field to be found")
	}
	if value != "default_value" {
		t.Errorf("want value %q, got %q", "default_value", value)
	}
}

func TestFieldOptionConstructor_FromContext(t *testing.T) {
	type contextKey struct{}
	ctx := context.Background()
	ctx = context.WithValue(ctx, contextKey{}, "context_value")

	constructor, extractor := errdef.DefineField[string]("test_field")
	fromContextConstructor := constructor.FromContext(func(ctx context.Context) string {
		return ctx.Value(contextKey{}).(string)
	})

	option := fromContextConstructor(ctx)
	err := errdef.New("test error", option)

	value, found := extractor(err)
	if !found {
		t.Error("want field to be found")
	}
	if value != "context_value" {
		t.Errorf("want value %q, got %q", "context_value", value)
	}
}

func TestFieldOptionExtractor_SingleReturn(t *testing.T) {
	constructor, extractor := errdef.DefineField[string]("test_field")
	singleReturnExtractor := extractor.SingleReturn()

	option := constructor("test_value")
	err := errdef.New("test error", option)

	value := singleReturnExtractor(err)
	if value != "test_value" {
		t.Errorf("want value %q, got %q", "test_value", value)
	}

	regularErr := errors.New("regular error")
	zeroValue := singleReturnExtractor(regularErr)
	if zeroValue != "" {
		t.Errorf("want zero value for string, got %q", zeroValue)
	}
}

func TestFieldOptionExtractor_SingleReturnDefault(t *testing.T) {
	constructor, extractor := errdef.DefineField[string]("test_field")
	singleReturnDefaultExtractor := extractor.SingleReturnDefault("default")

	option := constructor("test_value")
	err := errdef.New("test error", option)

	value := singleReturnDefaultExtractor(err)
	if value != "test_value" {
		t.Errorf("want value %q, got %q", "test_value", value)
	}

	regularErr := errors.New("regular error")
	defaultValue := singleReturnDefaultExtractor(regularErr)
	if defaultValue != "default" {
		t.Errorf("want default value %q, got %q", "default", defaultValue)
	}
}

func TestFieldOptionExtractor_SingleReturnDefaultFunc(t *testing.T) {
	constructor, extractor := errdef.DefineField[string]("test_field")
	singleReturnDefaultFuncExtractor := extractor.SingleReturnDefaultFunc(func(err error) string {
		return err.Error() + " default"
	})

	option := constructor("test_value")
	err := errdef.New("test error", option)

	value := singleReturnDefaultFuncExtractor(err)
	if value != "test_value" {
		t.Errorf("want value %q, got %q", "test_value", value)
	}

	regularErr := errors.New("regular error")
	defaultValue := singleReturnDefaultFuncExtractor(regularErr)
	if defaultValue != "regular error default" {
		t.Errorf("want default value %q, got %q", "regular error default", defaultValue)
	}
}
