package mathexpr

import (
	"strings"
	"testing"
)

const benchRuns = 100

// The zero-allocation guarantee is part of the API contract, so it is
// enforced by a test (fails CI), not just observed in benchmark output.
func TestZeroAllocs(t *testing.T) {
	for name, in := range benchInputs() {
		if n := testing.AllocsPerRun(benchRuns, func() { _ = Valid(in) }); n != 0 {
			t.Errorf("Valid(%s) allocates %.0f times per run, want 0", name, n)
		}
	}
	if n := testing.AllocsPerRun(benchRuns, func() { _ = Validate("(12.5-3)+(4-(-2.75))") }); n != 0 {
		t.Errorf("Validate on valid input allocates %.0f times per run, want 0", n)
	}
}

func BenchmarkValid(b *testing.B) {
	for name, in := range benchInputs() {
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(in)))
			for b.Loop() {
				Valid(in)
			}
		})
	}
}

// The failing path of Validate allocates exactly one error; the passing
// path must stay at zero.
func BenchmarkValidatePass(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		if Validate("(12.5-3)+(4-(-2.75))") != nil {
			b.Fatal("unexpected error")
		}
	}
}

// Reference oracle benchmarked for the README comparison table.
func BenchmarkReferenceParser(b *testing.B) {
	for name, in := range benchInputs() {
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(in)))
			for b.Loop() {
				refValid(in)
			}
		})
	}
}

func benchInputs() map[string]string {
	long := "1" + strings.Repeat("+1", 10_000)

	var nested strings.Builder
	const depth = 10_000
	for range depth {
		nested.WriteByte('(')
	}
	nested.WriteByte('1')
	for range depth {
		nested.WriteByte(')')
	}

	return map[string]string{
		"short":   "1+2",
		"typical": "(12.5-3)+(4-(-2.75))",
		"long":    long,
		"nested":  nested.String(),
	}
}
