package unmarshaler_test

import (
	"encoding/xml"
	"testing"

	"github.com/shiwano/errdef"
	"github.com/shiwano/errdef/unmarshaler"
)

type (
	xmlDecodedData struct {
		XMLName xml.Name       `xml:"error"`
		Message string         `xml:"message"`
		Kind    errdef.Kind    `xml:"kind"`
		Fields  map[string]any `xml:"fields"`
		Stack   []xmlFrame     `xml:"stack>frame"`
		Causes  []xmlCauseData `xml:"causes>cause"`
	}

	xmlFrame struct {
		Func string `xml:"func"`
		File string `xml:"file"`
		Line int    `xml:"line"`
	}

	xmlCauseData struct {
		Message string         `xml:"message"`
		Kind    string         `xml:"kind"`
		Type    string         `xml:"type"`
		Fields  map[string]any `xml:"fields"`
	}
)

func xmlDecoder(data []byte) (*unmarshaler.DecodedData, error) {
	var xmlData xmlDecodedData
	if err := xml.Unmarshal(data, &xmlData); err != nil {
		return nil, err
	}

	decoded := &unmarshaler.DecodedData{
		Message: xmlData.Message,
		Kind:    xmlData.Kind,
		Fields:  xmlData.Fields,
	}

	if len(xmlData.Stack) > 0 {
		decoded.Stack = make([]errdef.Frame, len(xmlData.Stack))
		for i, f := range xmlData.Stack {
			decoded.Stack[i] = errdef.Frame{
				Func: f.Func,
				File: f.File,
				Line: f.Line,
			}
		}
	}

	if len(xmlData.Causes) > 0 {
		decoded.Causes = make([]map[string]any, len(xmlData.Causes))
		for i, c := range xmlData.Causes {
			cause := map[string]any{
				"message": c.Message,
				"kind":    c.Kind,
			}
			if c.Type != "" {
				cause["type"] = c.Type
			}
			if c.Fields != nil {
				cause["fields"] = c.Fields
			}
			decoded.Causes[i] = cause
		}
	}

	return decoded, nil
}

func TestXMLDecoder_BasicUnmarshal(t *testing.T) {
	def := errdef.Define("test_error")
	resolver := errdef.NewResolver(def)
	u := unmarshaler.New(resolver, xmlDecoder)

	xmlData := `<error>
		<message>test message</message>
		<kind>test_error</kind>
	</error>`

	unmarshaled, err := u.Unmarshal([]byte(xmlData))
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if unmarshaled.Error() != "test message" {
		t.Errorf("want message %q, got %q", "test message", unmarshaled.Error())
	}
	if unmarshaled.Kind() != "test_error" {
		t.Errorf("want kind %q, got %q", "test_error", unmarshaled.Kind())
	}
}

func TestXMLDecoder_WithFields(t *testing.T) {
	userID, userIDFrom := errdef.DefineField[int]("user_id")
	def := errdef.Define("test_error", userID(0))
	resolver := errdef.NewResolver(def)
	u := unmarshaler.New(resolver, xmlDecoder)

	xmlData := `<error>
		<message>user not found</message>
		<kind>test_error</kind>
	</error>`

	unmarshaled, err := u.Unmarshal([]byte(xmlData))
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if unmarshaled.Error() != "user not found" {
		t.Errorf("want message %q, got %q", "user not found", unmarshaled.Error())
	}

	if _, ok := userIDFrom(unmarshaled); ok {
		t.Error("want user_id field to not be set")
	}
}

func TestXMLDecoder_WithCauses(t *testing.T) {
	def := errdef.Define("outer_error")
	innerDef := errdef.Define("inner_error")
	resolver := errdef.NewResolver(def, innerDef)
	u := unmarshaler.New(resolver, xmlDecoder)

	xmlData := `<error>
		<message>outer message</message>
		<kind>outer_error</kind>
		<causes>
			<cause>
				<message>inner message</message>
				<kind>inner_error</kind>
			</cause>
		</causes>
	</error>`

	unmarshaled, err := u.Unmarshal([]byte(xmlData))
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	causes := unmarshaled.Unwrap()
	if len(causes) != 1 {
		t.Fatalf("want 1 cause, got %d", len(causes))
	}

	if causes[0].Error() != "inner message" {
		t.Errorf("want cause message %q, got %q", "inner message", causes[0].Error())
	}

	if causeErr, ok := causes[0].(errdef.Error); ok {
		if causeErr.Kind() != "inner_error" {
			t.Errorf("want cause kind %q, got %q", "inner_error", causeErr.Kind())
		}
	} else {
		t.Error("want cause to be errdef.Error")
	}
}

func TestXMLDecoder_WithMultipleCauses(t *testing.T) {
	def := errdef.Define("outer_error")
	inner1Def := errdef.Define("inner1_error")
	inner2Def := errdef.Define("inner2_error")
	resolver := errdef.NewResolver(def, inner1Def, inner2Def)
	u := unmarshaler.New(resolver, xmlDecoder)

	xmlData := `<error>
		<message>outer message</message>
		<kind>outer_error</kind>
		<causes>
			<cause>
				<message>inner message 1</message>
				<kind>inner1_error</kind>
			</cause>
			<cause>
				<message>inner message 2</message>
				<kind>inner2_error</kind>
			</cause>
		</causes>
	</error>`

	unmarshaled, err := u.Unmarshal([]byte(xmlData))
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	causes := unmarshaled.Unwrap()
	if len(causes) != 2 {
		t.Fatalf("want 2 causes, got %d", len(causes))
	}

	if causes[0].Error() != "inner message 1" {
		t.Errorf("want first cause message %q, got %q", "inner message 1", causes[0].Error())
	}
	if causes[1].Error() != "inner message 2" {
		t.Errorf("want second cause message %q, got %q", "inner message 2", causes[1].Error())
	}
}

func TestXMLDecoder_UnknownCauseError(t *testing.T) {
	def := errdef.Define("test_error")
	resolver := errdef.NewResolver(def)
	u := unmarshaler.New(resolver, xmlDecoder)

	xmlData := `<error>
		<message>outer message</message>
		<kind>test_error</kind>
		<causes>
			<cause>
				<message>unknown error</message>
				<type>CustomError</type>
			</cause>
		</causes>
	</error>`

	unmarshaled, err := u.Unmarshal([]byte(xmlData))
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	causes := unmarshaled.Unwrap()
	if len(causes) != 1 {
		t.Fatalf("want 1 cause, got %d", len(causes))
	}

	if causes[0].Error() != "unknown error" {
		t.Errorf("want cause message %q, got %q", "unknown error", causes[0].Error())
	}
}
