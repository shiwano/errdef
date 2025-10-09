package unmarshaler_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/shiwano/errdef"
	"github.com/shiwano/errdef/resolver"
	"github.com/shiwano/errdef/unmarshaler"
)

var (
	benchDef                   = errdef.Define("benchmark_error", errdef.HTTPStatus(500))
	benchField, _              = errdef.DefineField[string]("bench_field")
	benchResolver              = resolver.New(benchDef)
	benchUnmarshaler           = unmarshaler.NewJSON(benchResolver)
	benchUnmarshalerWithStdlib = unmarshaler.NewJSON(benchResolver, unmarshaler.WithStandardSentinelErrors())
)

// unmarshaler: Unmarshal simple error
func BenchmarkUnmarshalerUnmarshalSimple(b *testing.B) {
	err := benchDef.New("benchmark error")
	data, _ := json.Marshal(err.(errdef.Error))
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_, _ = benchUnmarshaler.Unmarshal(data)
	}
}

// unmarshaler: Unmarshal error with fields
func BenchmarkUnmarshalerUnmarshalWithFields(b *testing.B) {
	err := benchDef.WithOptions(benchField("test_value")).New("benchmark error")
	data, _ := json.Marshal(err.(errdef.Error))
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_, _ = benchUnmarshaler.Unmarshal(data)
	}
}

// unmarshaler: Unmarshal error with shallow chain (3 levels)
func BenchmarkUnmarshalerUnmarshalShallowChain(b *testing.B) {
	err1 := benchDef.New("level 1")
	err2 := benchDef.Wrap(err1)
	err3 := benchDef.Wrap(err2)
	data, _ := json.Marshal(err3.(errdef.Error))
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_, _ = benchUnmarshaler.Unmarshal(data)
	}
}

// unmarshaler: Unmarshal error with deep chain (10 levels)
func BenchmarkUnmarshalerUnmarshalDeepChain(b *testing.B) {
	err := benchDef.New("level 1")
	for range 9 {
		err = benchDef.Wrap(err)
	}
	data, _ := json.Marshal(err.(errdef.Error))
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_, _ = benchUnmarshaler.Unmarshal(data)
	}
}

// unmarshaler: Unmarshal with standard sentinel error
func BenchmarkUnmarshalerUnmarshalWithStdlibError(b *testing.B) {
	err := benchDef.Wrap(errors.New("stdlib error"))
	data, _ := json.Marshal(err.(errdef.Error))
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_, _ = benchUnmarshalerWithStdlib.Unmarshal(data)
	}
}

// unmarshaler: Round-trip (Marshal + Unmarshal)
func BenchmarkUnmarshalerRoundTrip(b *testing.B) {
	err := benchDef.WithOptions(benchField("test_value")).New("benchmark error")
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		data, _ := json.Marshal(err.(errdef.Error))
		_, _ = benchUnmarshaler.Unmarshal(data)
	}
}

// unmarshaler: Round-trip with deep chain (10 levels)
func BenchmarkUnmarshalerRoundTripDeepChain(b *testing.B) {
	err := benchDef.New("level 1")
	for range 9 {
		err = benchDef.Wrap(err)
	}
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		data, _ := json.Marshal(err.(errdef.Error))
		_, _ = benchUnmarshaler.Unmarshal(data)
	}
}
