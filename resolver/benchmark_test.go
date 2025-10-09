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

// resolver: ResolveKindStrict with 3 definitions
func BenchmarkResolverResolveKindStrict3First(b *testing.B) {
	r := resolver.New(benchDef1, benchDef2, benchDef3)
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_, _ = r.ResolveKindStrict("error_1")
	}
}

// resolver: ResolveFieldStrict with 3 definitions (first match)
func BenchmarkResolverResolveFieldStrict3First(b *testing.B) {
	r := resolver.New(benchDef1, benchDef2, benchDef3)
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_, _ = r.ResolveFieldStrict(errdef.HTTPStatus.Key(), 400)
	}
}

// resolver: ResolveFieldStrict with 3 definitions (last match)
func BenchmarkResolverResolveFieldStrict3Last(b *testing.B) {
	r := resolver.New(benchDef1, benchDef2, benchDef3)
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_, _ = r.ResolveFieldStrict(errdef.HTTPStatus.Key(), 500)
	}
}

// resolver: ResolveFieldStrict with 3 definitions (not found)
func BenchmarkResolverResolveFieldStrict3NotFound(b *testing.B) {
	r := resolver.New(benchDef1, benchDef2, benchDef3)
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_, _ = r.ResolveFieldStrict(errdef.HTTPStatus.Key(), 999)
	}
}

// resolver: FallbackResolver ResolveKind with fallback
func BenchmarkFallbackResolverResolveKind(b *testing.B) {
	fallbackDef := errdef.Define("fallback")
	r := resolver.New(benchDef1, benchDef2, benchDef3).WithFallback(fallbackDef)
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_ = r.ResolveKind("not_found")
	}
}
