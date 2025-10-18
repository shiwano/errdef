package resolver_test

import (
	"testing"

	"github.com/shiwano/errdef"
	"github.com/shiwano/errdef/resolver"
)

var (
	benchDef1 = errdef.Define("error_1", errdef.HTTPStatus(400))
	benchDef2 = errdef.Define("error_2", errdef.HTTPStatus(404))
	benchDef3 = errdef.Define("error_3", errdef.HTTPStatus(500))
)

// resolver: ResolveKind with 3 definitions
func BenchmarkResolverResolveKind3First(b *testing.B) {
	r := resolver.New(benchDef1, benchDef2, benchDef3)
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_, _ = r.ResolveKind("error_1")
	}
}

// resolver: ResolveField with 3 definitions (first match)
func BenchmarkResolverResolveField3First(b *testing.B) {
	r := resolver.New(benchDef1, benchDef2, benchDef3)
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_, _ = r.ResolveField(errdef.HTTPStatus.Key(), 400)
	}
}

// resolver: ResolveField with 3 definitions (last match)
func BenchmarkResolverResolveField3Last(b *testing.B) {
	r := resolver.New(benchDef1, benchDef2, benchDef3)
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_, _ = r.ResolveField(errdef.HTTPStatus.Key(), 500)
	}
}

// resolver: ResolveField with 3 definitions (not found)
func BenchmarkResolverResolveField3NotFound(b *testing.B) {
	r := resolver.New(benchDef1, benchDef2, benchDef3)
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_, _ = r.ResolveField(errdef.HTTPStatus.Key(), 999)
	}
}

// resolver: DefaultResolver ResolveKindOrDefault with default
func BenchmarkDefaultResolverResolveKindOrDefault(b *testing.B) {
	defaultDef := errdef.Define("default")
	r := resolver.New(benchDef1, benchDef2, benchDef3).WithDefault(defaultDef)
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_ = r.ResolveKindOrDefault("not_found")
	}
}
