// Command example demonstrates the library API.
package main

import (
	"fmt"
	"testing"

	mathexpr "github.com/incu6us/math-expressions-validation"
)

const allocRuns = 1000

type expr struct {
	val     string
	isValid bool
}

func main() {
	exprs := []expr{
		{
			val:     "(12.5-3)+(4-(-2.75))",
			isValid: true,
		},
		{
			val:     "1++2",
			isValid: false,
		},
		{
			val:     "1+*2",
			isValid: false,
		},
		{
			val:     "(1+2",
			isValid: false,
		},
	}

	for _, e := range exprs {
		fmt.Printf("Valid(%s) = %t\n", e.val, mathexpr.Valid(e.val))
		if err := mathexpr.Validate(e.val); err != nil {
			fmt.Println("  ", err)
		}
	}

	validAllocs := testing.AllocsPerRun(allocRuns, func() {
		for _, e := range exprs {
			_ = mathexpr.Valid(e.val)
		}
	})

	validateAllocs := testing.AllocsPerRun(allocRuns, func() {
		for _, e := range exprs {
			_ = mathexpr.Validate(e.val)
		}
	})

	fmt.Printf("\nheap allocations over all %d inputs (%d invalid):\n", len(exprs), countInvalid(exprs))
	fmt.Printf("  Valid:    %.0f\n", validAllocs)
	fmt.Printf("  Validate: %.0f (one *SyntaxError per invalid input (allocates only on error))\n", validateAllocs)
}

func countInvalid(exprs []expr) int {
	var invalid int
	for _, e := range exprs {
		if e.isValid {
			continue
		}
		invalid++
	}
	return invalid
}
