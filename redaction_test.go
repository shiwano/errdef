package errdef

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log/slog"
	"strings"
	"testing"
)

func TestRedact(t *testing.T) {
	t.Run("string value", func(t *testing.T) {
		secret := "my-secret-password"
		redacted := Redact(secret)

		if redacted.Value() != secret {
			t.Errorf("want value %v, got %v", secret, redacted.Value())
		}
	})

	t.Run("int value", func(t *testing.T) {
		secret := 12345
		redacted := Redact(secret)

		if redacted.Value() != secret {
			t.Errorf("want value %v, got %v", secret, redacted.Value())
		}
	})

	t.Run("struct value", func(t *testing.T) {
		type credentials struct {
			Username string
			Password string
		}
		secret := credentials{Username: "admin", Password: "secret"}
		redacted := Redact(secret)

		if redacted.Value() != secret {
			t.Errorf("want value %v, got %v", secret, redacted.Value())
		}
	})
}

func TestRedacted_String(t *testing.T) {
	redacted := Redact("secret")
	want := "[REDACTED]"
	got := redacted.String()

	if got != want {
		t.Errorf("want %v, got %v", want, got)
	}
}

func TestRedacted_GoString(t *testing.T) {
	redacted := Redact("secret")
	want := "[REDACTED]"
	got := redacted.GoString()

	if got != want {
		t.Errorf("want %v, got %v", want, got)
	}
}

func TestRedacted_Format(t *testing.T) {
	redacted := Redact("secret")

	tests := []struct {
		name   string
		format string
		want   string
	}{
		{"with %s", "%s", "[REDACTED]"},
		{"with %v", "%v", "[REDACTED]"},
		{"with %+v", "%+v", "[REDACTED]"},
		{"with %#v", "%#v", "[REDACTED]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fmt.Sprintf(tt.format, redacted)
			if got != tt.want {
				t.Errorf("want %v, got %v", tt.want, got)
			}
		})
	}
}

func TestRedacted_MarshalJSON(t *testing.T) {
	type container struct {
		Secret Redacted[string] `json:"secret"`
	}

	c := container{Secret: Redact("my-secret")}
	got, err := json.Marshal(c)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	want := `{"secret":"[REDACTED]"}`
	if string(got) != want {
		t.Errorf("want %v, got %v", want, string(got))
	}
}

func TestRedacted_MarshalText(t *testing.T) {
	type container struct {
		Secret Redacted[string] `xml:"secret"`
	}

	c := container{Secret: Redact("my-secret")}
	got, err := xml.Marshal(c)
	if err != nil {
		t.Fatalf("xml.Marshal() error = %v", err)
	}

	want := `<container><secret>[REDACTED]</secret></container>`
	if string(got) != want {
		t.Errorf("want %v, got %v", want, string(got))
	}
}

func TestRedacted_MarshalBinary(t *testing.T) {
	type container struct {
		Secret Redacted[string]
	}

	c := container{Secret: Redact("my-secret-password")}

	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(&c)
	if err != nil {
		t.Fatalf("gob.Encode() error = %v", err)
	}

	encodedData := buf.String()
	if strings.Contains(encodedData, "my-secret-password") {
		t.Error("want secret to be redacted, but found in gob encoded data")
	}
	if !strings.Contains(encodedData, "[REDACTED]") {
		t.Error("want [REDACTED] in gob encoded data")
	}
}

func TestRedacted_LogValue(t *testing.T) {
	t.Run("log value", func(t *testing.T) {
		redacted := Redact("secret")
		want := slog.StringValue("[REDACTED]")
		got := redacted.LogValue()

		if got.String() != want.String() {
			t.Errorf("want %v, got %v", want, got)
		}
	})

	t.Run("actual logging", func(t *testing.T) {
		var buf bytes.Buffer
		logger := slog.New(slog.NewJSONHandler(&buf, nil))

		ctor, _ := DefineField[Redacted[string]]("password")
		def := Define("auth_error", ctor(Redact("my-secret-password")))
		err := def.New("authentication failed")

		logger.Error("error occurred", "error", err)

		output := buf.String()
		if strings.Contains(output, "my-secret-password") {
			t.Error("want secret to be redacted, but found in log output")
		}
		if !strings.Contains(output, "\"password\":\"[REDACTED]\"") {
			t.Error("want [REDACTED] in log output")
		}
	})
}
