package errdef_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/shiwano/errdef"
)

func TestPanicError_Error(t *testing.T) {
	def := errdef.Define("test_error")
	panicValue := errors.New("panic error")
	var err error
	def.CapturePanic(&err, panicValue)

	var panicErr errdef.PanicError
	if !errors.As(err, &panicErr) {
		t.Fatal("want error to be a PanicError")
	}

	if panicErr.Error() != "panic error" {
		t.Errorf("want error message %q, got %q", "panic error", panicErr.Error())
	}
}

func TestPanicError_PanicValue(t *testing.T) {
	def := errdef.Define("test_error")
	panicValue := errors.New("panic error")
	var err error
	def.CapturePanic(&err, panicValue)

	var panicErr errdef.PanicError
	if !errors.As(err, &panicErr) {
		t.Fatal("want error to be a PanicError")
	}

	if panicErr.PanicValue() != panicValue {
		t.Errorf("want panic value %v, got %v", panicValue, panicErr.PanicValue())
	}
}

func TestPanicError_Unwrap(t *testing.T) {
	t.Run("with error", func(t *testing.T) {
		def := errdef.Define("test_error")
		panicValue := errors.New("panic error")
		var err error
		def.CapturePanic(&err, panicValue)

		var panicErr errdef.PanicError
		if !errors.As(err, &panicErr) {
			t.Fatal("want error to be a PanicError")
		}

		if panicErr.Unwrap() != panicValue {
			t.Errorf("want unwrapped error %v, got %v", panicValue, panicErr.Unwrap())
		}
	})

	t.Run("with non error", func(t *testing.T) {
		def := errdef.Define("test_error")
		panicValue := "panic string"
		var err error
		def.CapturePanic(&err, panicValue)

		var panicErr errdef.PanicError
		if !errors.As(err, &panicErr) {
			t.Fatal("want error to be a PanicError")
		}

		if panicErr.Unwrap() != nil {
			t.Errorf("want unwrapped error nil, got %v", panicErr.Unwrap())
		}
	})
}

func TestPanicError_Format(t *testing.T) {
	type testStruct struct {
		Name  string
		Value int
	}

	tests := []struct {
		name       string
		panicValue any
		formats    map[string]string
	}{
		{
			name:       "with standard error",
			panicValue: errors.New("panic error"),
			formats: map[string]string{
				"%v":  "panic error",
				"%s":  "panic error",
				"%+v": "panic error\n---\npanic_value: panic error",
				"%q":  `"panic error"`,
			},
		},
		{
			name:       "with string value",
			panicValue: "panic string",
			formats: map[string]string{
				"%v":  "panic string",
				"%s":  "panic string",
				"%#v": `&errdef.panicError{msg:"panic string", panicValue:"panic string"}`,
				"%+v": "panic string\n---\npanic_value: panic string",
				"%q":  `"panic string"`,
			},
		},
		{
			name:       "with integer value",
			panicValue: 42,
			formats: map[string]string{
				"%v":  "42",
				"%s":  "42",
				"%#v": `&errdef.panicError{msg:"42", panicValue:42}`,
				"%+v": "42\n---\npanic_value: 42",
				"%q":  `"42"`,
			},
		},
		{
			name:       "with struct value",
			panicValue: testStruct{Name: "test", Value: 123},
			formats: map[string]string{
				"%v":  "{test 123}",
				"%s":  "{test 123}",
				"%#v": `&errdef.panicError{msg:"{test 123}", panicValue:errdef_test.testStruct{Name:"test", Value:123}}`,
				"%+v": "{test 123}\n---\npanic_value: {Name:test Value:123}",
				"%q":  `"{test 123}"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			def := errdef.Define("test_error")
			var err error
			def.CapturePanic(&err, tt.panicValue)

			var panicErr errdef.PanicError
			if !errors.As(err, &panicErr) {
				t.Fatal("want error to be a PanicError")
			}

			for format, want := range tt.formats {
				got := fmt.Sprintf(format, panicErr)
				if got != want {
					t.Errorf("%s: want %q, got %q", format, want, got)
				}
			}
		})
	}
}
