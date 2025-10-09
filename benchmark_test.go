package errdef_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"testing"

	"github.com/shiwano/errdef"
)

var (
	benchDef                        = errdef.Define("benchmark_error")
	benchDefNoTrace                 = errdef.Define("benchmark_error_notrace", errdef.NoTrace())
	benchDefStackDepth              = errdef.Define("benchmark_error_stackdepth", errdef.StackDepth(1))
	benchField, benchFieldExtractor = errdef.DefineField[string]("bench_field")
	benchField1, _                  = errdef.DefineField[string]("bench_field_1")
	benchField2, _                  = errdef.DefineField[string]("bench_field_2")
	benchField3, _                  = errdef.DefineField[string]("bench_field_3")
	benchField4, _                  = errdef.DefineField[string]("bench_field_4")
	benchField5, _                  = errdef.DefineField[string]("bench_field_5")
	benchField6, _                  = errdef.DefineField[string]("bench_field_6")
	benchField7, _                  = errdef.DefineField[string]("bench_field_7")
	benchField8, _                  = errdef.DefineField[string]("bench_field_8")
	benchField9, _                  = errdef.DefineField[string]("bench_field_9")
	benchField10, _                 = errdef.DefineField[string]("bench_field_10")
)

// Baseline: Standard library error creation
func BenchmarkStdlibNew(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = errors.New("benchmark error")
	}
}

// Baseline: Standard library error wrapping
func BenchmarkStdlibWrap(b *testing.B) {
	cause := errors.New("cause error")
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_ = fmt.Errorf("wrapped: %w", cause)
	}
}

// errdef: New with default stack trace
func BenchmarkErrdefNew(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = benchDef.New("benchmark error")
	}
}

// errdef: New with NoTrace option
func BenchmarkErrdefNewNoTrace(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = benchDefNoTrace.New("benchmark error")
	}
}

// errdef: Wrap with default stack trace
func BenchmarkErrdefWrap(b *testing.B) {
	cause := errors.New("cause error")
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_ = benchDef.Wrap(cause)
	}
}

// errdef: Wrap with NoTrace option
func BenchmarkErrdefWrapNoTrace(b *testing.B) {
	cause := errors.New("cause error")
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_ = benchDefNoTrace.Wrap(cause)
	}
}

// errdef: Creating error with fields
func BenchmarkErrdefWithFields(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = benchDef.WithOptions(benchField("test_value")).New("benchmark error")
	}
}

// errdef: Creating error with fields and NoTrace
func BenchmarkErrdefWithFieldsNoTrace(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = benchDefNoTrace.WithOptions(benchField("test_value")).New("benchmark error")
	}
}

// errdef: WithOptions with 3 fields
func BenchmarkErrdefWithOptions3Fields(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = benchDef.WithOptions(
			benchField1("value1"),
			benchField2("value2"),
			benchField3("value3"),
		).New("benchmark error")
	}
}

// errdef: WithOptions with 5 fields
func BenchmarkErrdefWithOptions5Fields(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = benchDef.WithOptions(
			benchField1("value1"),
			benchField2("value2"),
			benchField3("value3"),
			benchField4("value4"),
			benchField5("value5"),
		).New("benchmark error")
	}
}

// errdef: WithOptions with 10 fields
func BenchmarkErrdefWithOptions10Fields(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = benchDef.WithOptions(
			benchField1("value1"),
			benchField2("value2"),
			benchField3("value3"),
			benchField4("value4"),
			benchField5("value5"),
			benchField6("value6"),
			benchField7("value7"),
			benchField8("value8"),
			benchField9("value9"),
			benchField10("value10"),
		).New("benchmark error")
	}
}

// errdef: Field extraction
func BenchmarkErrdefFieldExtraction(b *testing.B) {
	err := benchDef.WithOptions(benchField("test_value")).New("benchmark error")
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_, _ = benchFieldExtractor(err)
	}
}

// errdef: Error chain unwrapping (shallow: 3 levels)
func BenchmarkErrdefUnwrapTreeShallow(b *testing.B) {
	err1 := benchDef.New("level 1")
	err2 := benchDef.Wrap(err1)
	err3 := benchDef.Wrap(err2)
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_ = err3.(errdef.Error).UnwrapTree()
	}
}

// errdef: Error chain unwrapping (deep: 10 levels)
func BenchmarkErrdefUnwrapTreeDeep(b *testing.B) {
	err := benchDef.New("level 1")
	for range 9 {
		err = benchDef.Wrap(err)
	}
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_ = err.(errdef.Error).UnwrapTree()
	}
}

// errdef: JSON marshaling (default stack)
func BenchmarkErrdefJSONMarshal(b *testing.B) {
	err := benchDef.WithOptions(benchField("test_value")).New("benchmark error")
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_, _ = err.(errdef.Error).(json.Marshaler).MarshalJSON()
	}
}

// errdef: JSON marshaling (NoTrace)
func BenchmarkErrdefJSONMarshalNoTrace(b *testing.B) {
	err := benchDefNoTrace.WithOptions(benchField("test_value")).New("benchmark error")
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_, _ = err.(errdef.Error).(json.Marshaler).MarshalJSON()
	}
}

// errdef: Format with %+v (detailed output)
func BenchmarkErrdefFormatDetailed(b *testing.B) {
	err := benchDef.WithOptions(benchField("test_value")).New("benchmark error")
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_ = fmt.Sprintf("%+v", err)
	}
}

// errdef: slog.LogValue
func BenchmarkErrdefLogValue(b *testing.B) {
	err := benchDef.WithOptions(benchField("test_value")).New("benchmark error")
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_ = err.(slog.LogValuer).LogValue()
	}
}

// errdef: New with StackDepth(1)
func BenchmarkErrdefNewStackDepth1(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = benchDefStackDepth.New("benchmark error")
	}
}

// errdef: Wrap with StackDepth(1)
func BenchmarkErrdefWrapStackDepth1(b *testing.B) {
	cause := errors.New("cause error")
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_ = benchDefStackDepth.Wrap(cause)
	}
}

// errdef: JSON marshal deep chain without Boundary at all (10 levels)
func BenchmarkErrdefJSONMarshalDeepChainNoBoundary(b *testing.B) {
	err := benchDef.New("level 1")
	for range 9 {
		err = benchDef.Wrap(err)
	}
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_, _ = err.(errdef.Error).(json.Marshaler).MarshalJSON()
	}
}

// errdef: JSON marshal deep chain with Boundary at level 3 (10 levels total)
func BenchmarkErrdefJSONMarshalDeepChainWithBoundary(b *testing.B) {
	err := benchDef.New("level 1")
	for range 2 {
		err = benchDef.Wrap(err)
	}
	boundaryDef := errdef.Define("boundary", errdef.Boundary())
	err = boundaryDef.Wrap(err)
	for range 6 {
		err = benchDef.Wrap(err)
	}
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_, _ = err.(errdef.Error).(json.Marshaler).MarshalJSON()
	}
}
