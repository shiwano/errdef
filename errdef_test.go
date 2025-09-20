package errdef_test

import (
	"errors"
	"testing"

	"github.com/shiwano/errdef"
)

func TestDefine(t *testing.T) {
	t.Run("basic kind", func(t *testing.T) {
		kind := errdef.Kind("test_error")
		def := errdef.Define(kind)

		if def.Kind() != kind {
			t.Errorf("want kind %v, got %v", kind, def.Kind())
		}

		if def.Error() != string(kind) {
			t.Errorf("want error string %q, got %q", string(kind), def.Error())
		}
	})

	t.Run("empty kind", func(t *testing.T) {
		def := errdef.Define("")

		if def.Kind() != "" {
			t.Errorf("want empty kind, got %v", def.Kind())
		}

		if def.Error() != "[unnamed]" {
			t.Errorf("want error string %q, got %q", "[unnamed]", def.Error())
		}
	})
}

func TestDefineField(t *testing.T) {
	t.Run("constructor and extractor", func(t *testing.T) {
		constructor, extractor := errdef.DefineField[string]("test_field")

		option := constructor("test_value")
		err := errdef.New("test error", option)

		value, found := extractor(err)
		if !found {
			t.Error("want field to be found")
		}
		if value != "test_value" {
			t.Errorf("want value %q, got %q", "test_value", value)
		}
	})

	t.Run("extractor with wrong value type", func(t *testing.T) {
		type valueType string

		constructor, _ := errdef.DefineField[string]("test_field")
		_, extractor := errdef.DefineField[valueType]("test_field")

		option := constructor("test_value")
		err := errdef.New("test error", option)

		_, found := extractor(err)
		if found {
			t.Error("want field not to be found with wrong type")
		}
	})

	t.Run("extractor with wrong key type", func(t *testing.T) {
		constructor, _ := errdef.DefineField[string]("test_field")
		_, extractor := errdef.DefineField[string]("test_field")

		option := constructor("test_value")
		err := errdef.New("test error", option)

		_, found := extractor(err)
		if found {
			t.Error("want field not to be found with wrong type")
		}
	})

	t.Run("extractor on non-errdef error", func(t *testing.T) {
		_, extractor := errdef.DefineField[string]("test_field")

		err := errors.New("regular error")

		_, found := extractor(err)
		if found {
			t.Error("want field not to be found on non-errdef error")
		}
	})
}

func TestNew(t *testing.T) {
	t.Run("basic error creation", func(t *testing.T) {
		err := errdef.New("test message")

		if err.Error() != "test message" {
			t.Errorf("want message %q, got %q", "test message", err.Error())
		}
	})

	t.Run("error creation with field option", func(t *testing.T) {
		constructor, extractor := errdef.DefineField[string]("user_id")

		err := errdef.New("test message", constructor("user123"))

		value, found := extractor(err)
		if !found {
			t.Error("want field to be found")
		}
		if value != "user123" {
			t.Errorf("want value %q, got %q", "user123", value)
		}
	})

	t.Run("error creation with multiple options", func(t *testing.T) {
		userIDConstructor, userIDExtractor := errdef.DefineField[string]("user_id")
		countConstructor, countExtractor := errdef.DefineField[int]("count")

		err := errdef.New("test message",
			userIDConstructor("user123"),
			countConstructor(42),
		)

		userID, found := userIDExtractor(err)
		if !found {
			t.Error("want user_id field to be found")
		}
		if userID != "user123" {
			t.Errorf("want user_id %q, got %q", "user123", userID)
		}

		count, found := countExtractor(err)
		if !found {
			t.Error("want count field to be found")
		}
		if count != 42 {
			t.Errorf("want count %d, got %d", 42, count)
		}
	})
}

func TestWrap(t *testing.T) {
	t.Run("basic error wrapping", func(t *testing.T) {
		original := errors.New("original error")
		wrapped := errdef.Wrap(original)

		if wrapped.Error() != "original error" {
			t.Errorf("want message %q, got %q", "original error", wrapped.Error())
		}

		if !errors.Is(wrapped, original) {
			t.Error("want wrapped error to be the original error")
		}
	})

	t.Run("wrapping nil error", func(t *testing.T) {
		wrapped := errdef.Wrap(nil)
		if wrapped != nil {
			t.Error("want nil when wrapping nil error")
		}
	})

	t.Run("wrapping with field option", func(t *testing.T) {
		constructor, extractor := errdef.DefineField[string]("context")
		original := errors.New("original error")

		wrapped := errdef.Wrap(original, constructor("test context"))

		if wrapped.Error() != "original error" {
			t.Errorf("want message %q, got %q", "original error", wrapped.Error())
		}

		value, found := extractor(wrapped)
		if !found {
			t.Error("want field to be found")
		}
		if value != "test context" {
			t.Errorf("want context %q, got %q", "test context", value)
		}

		if !errors.Is(wrapped, original) {
			t.Error("want wrapped error to be the original error")
		}
	})

	t.Run("wrapping with multiple options", func(t *testing.T) {
		userIDConstructor, userIDExtractor := errdef.DefineField[string]("user_id")
		attemptConstructor, attemptExtractor := errdef.DefineField[int]("attempt")
		original := errors.New("authentication failed")

		wrapped := errdef.Wrap(original,
			userIDConstructor("user456"),
			attemptConstructor(3))

		userID, found := userIDExtractor(wrapped)
		if !found {
			t.Error("want user_id field to be found")
		}
		if userID != "user456" {
			t.Errorf("want user_id %q, got %q", "user456", userID)
		}

		attempt, found := attemptExtractor(wrapped)
		if !found {
			t.Error("want attempt field to be found")
		}
		if attempt != 3 {
			t.Errorf("want attempt %d, got %d", 3, attempt)
		}

		if !errors.Is(wrapped, original) {
			t.Error("want wrapped error to be the original error")
		}
	})
}

func TestCapturePanic(t *testing.T) {
	t.Run("basic panic capture", func(t *testing.T) {
		var err error
		errdef.CapturePanic(&err, "test panic")

		if err == nil {
			t.Fatal("want error to be set")
		}

		if err.Error() != "panic: test panic" {
			t.Errorf("want error message %q, got %q", "panic: test panic", err.Error())
		}

		var panicErr errdef.PanicError
		if !errors.As(err, &panicErr) {
			t.Fatal("want error to be a PanicError")
		}

		if panicErr.PanicValue() != "test panic" {
			t.Errorf("want panic value %q, got %v", "test panic", panicErr.PanicValue())
		}
	})

	t.Run("capture with options", func(t *testing.T) {
		constructor, extractor := errdef.DefineField[string]("operation")
		var err error
		errdef.CapturePanic(&err, "service panic", constructor("database_query"))

		if err == nil {
			t.Fatal("want error to be set")
		}

		value, found := extractor(err)
		if !found {
			t.Error("want field to be found")
		}
		if value != "database_query" {
			t.Errorf("want field value %q, got %q", "database_query", value)
		}

		var panicErr errdef.PanicError
		if !errors.As(err, &panicErr) {
			t.Fatal("want error to be a PanicError")
		}

		if panicErr.PanicValue() != "service panic" {
			t.Errorf("want panic value %q, got %v", "service panic", panicErr.PanicValue())
		}
	})

	t.Run("capture with multiple options", func(t *testing.T) {
		userIDConstructor, userIDExtractor := errdef.DefineField[string]("user_id")
		serviceConstructor, serviceExtractor := errdef.DefineField[string]("service")
		var err error

		errdef.CapturePanic(&err, "handler panic",
			userIDConstructor("user789"),
			serviceConstructor("auth_service"))

		if err == nil {
			t.Fatal("want error to be set")
		}

		userID, found := userIDExtractor(err)
		if !found {
			t.Error("want user_id field to be found")
		}
		if userID != "user789" {
			t.Errorf("want user_id %q, got %q", "user789", userID)
		}

		service, found := serviceExtractor(err)
		if !found {
			t.Error("want service field to be found")
		}
		if service != "auth_service" {
			t.Errorf("want service %q, got %q", "auth_service", service)
		}
	})

	t.Run("nil panic value", func(t *testing.T) {
		var err error
		errdef.CapturePanic(&err, nil)

		if err != nil {
			t.Errorf("want no error for nil panic value, got %v", err)
		}
	})

	t.Run("nil error pointer", func(t *testing.T) {
		errdef.CapturePanic(nil, "panic value")
	})

	t.Run("real panic scenario", func(t *testing.T) {
		var err error

		func() {
			defer func() {
				errdef.CapturePanic(&err, recover())
			}()
			panic("critical error")
		}()

		if err == nil {
			t.Fatal("want error to be set")
		}

		if err.Error() != "panic: critical error" {
			t.Errorf("want error message %q, got %q", "panic: critical error", err.Error())
		}

		var panicErr errdef.PanicError
		if !errors.As(err, &panicErr) {
			t.Fatal("want error to be a PanicError")
		}

		if panicErr.PanicValue() != "critical error" {
			t.Errorf("want panic value %q, got %v", "critical error", panicErr.PanicValue())
		}
	})
}
