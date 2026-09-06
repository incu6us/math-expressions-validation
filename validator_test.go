package mathexpr

import (
	"errors"
	"strings"
	"testing"
)

// Each table doubles as an executable specification of one grammar area.
func TestValid(t *testing.T) {
	type tc struct {
		expr string
		want bool
	}

	groups := map[string][]tc{
		"numbers": {
			{"0", true},
			{"7", true},
			{"42", true},
			{"3.14", true},
			{"0.5", true},
			{"123.456", true},
			{"007", true}, // leading zeros accepted (validator, not linter)
			{"3.", false},
			{".5", false},
			{"1.2.3", false},
			{"1..2", false},
			{"", false},
			{"   ", false},
			{"abc", false},
			{"1a", false},
		},
		"unary minus": {
			{"-5", true},
			{"-5.5", true},
			{"-(1+2)", true},
			{"(-5)", true},
			{"3-(-5)", true},
			{"-(-5)", true},
			{"--5", false},
			{"3+-5", false}, // unary after binary op rejected by design
			{"3--5", false},
			{"- 5", false}, // detached unary minus rejected by design
			{"-", false},
			{"+5", false}, // no unary plus
		},
		"binary operators": {
			{"1+2", true},
			{"1-2", true},
			{"1+2-3+4", true},
			{"1+", false},
			{"+", false},
			{"1++2", false},
			{"1+2+", false},
		},
		"parentheses": {
			{"(1)", true},
			{"(1+2)", true},
			{"((1+2))", true},
			{"(1+2)-(3+4)", true},
			{"((1+2)-3)+4", true},
			{"()", false},
			{"(", false},
			{")", false},
			{"(1+2", false},
			{"1+2)", false},
			{")(", false},
			{"(1))(2)", false},
			{"(-)", false},
			{"1+)", false},
			{"1.5)", false},
			{"(1.5)", true},
		},
		"whitespace": {
			{"1 + 2", true},
			{" 1+2 ", true},
			{"\t1\t-\t2\t", true},
			{"( 1 + 2 )", true},
			{"1.5 - 2", true},   // space ends a fractional number
			{"1  +  2", true},   // runs of whitespace between tokens
			{"(1+2) - 3", true}, // space after closing paren
			{"1 2", false},      // two values without operator
			{"1 . 5", false},    // space inside a number
			{"1. 5", false},
			{"1 .5", false},
		},
		"encoding": {
			{"1\n+2", false},     // only space and tab allowed
			{"1\u00a0+2", false}, // non-breaking space is not whitespace
			{"١+٢", false},       // non-ASCII digits invalid by construction
		},
	}

	for name, cases := range groups {
		t.Run(name, func(t *testing.T) {
			for _, c := range cases {
				if got := Valid(c.expr); got != c.want {
					t.Errorf("Valid(%q) = %t, want %t", c.expr, got, c.want)
				}
			}
		})
	}
}

func TestValidateReportsPosition(t *testing.T) {
	cases := []struct {
		expr    string
		wantPos int
	}{
		{"1+*2", 2},
		{"1+2)", 3},
		{"(1+2", 4}, // failure detected at end of input
		{"", 0},
		{"--5", 1},
	}
	for _, c := range cases {
		err := Validate(c.expr)
		if err == nil {
			t.Errorf("Validate(%q) = nil, want error", c.expr)
			continue
		}

		var syntaxErr *SyntaxError
		if !errors.As(err, &syntaxErr) {
			t.Errorf("Validate(%q) returned %T, want *SyntaxError", c.expr, err)
			continue
		}

		if syntaxErr.Pos != c.wantPos {
			t.Errorf("Validate(%q).Pos = %d, want %d", c.expr, syntaxErr.Pos, c.wantPos)
		}
	}
}

func TestSyntaxErrorMessage(t *testing.T) {
	cases := []struct {
		expr string
		want string
	}{
		{"1+*2", "mathexpr: unexpected character at position 2"},
		{")", "mathexpr: unbalanced parenthesis at position 0"},
		{"1+2)", "mathexpr: unbalanced parenthesis at position 3"},
		{"(1+2", "mathexpr: unbalanced parenthesis at position 4"},
		{"", "mathexpr: empty expression at position 0"},
		{"1+", "mathexpr: unexpected end of expression at position 2"},
	}
	for _, c := range cases {
		err := Validate(c.expr)
		if err == nil {
			t.Errorf("Validate(%q) = nil, want error", c.expr)
			continue
		}
		if got := err.Error(); got != c.want {
			t.Errorf("Validate(%q).Error() = %q, want %q", c.expr, got, c.want)
		}
	}
}

// reasonUnknown is unreachable through the public API by construction;
// exercise its message directly so the mapping is still pinned down.
func TestUnknownReasonMessage(t *testing.T) {
	if got := reasonUnknown.message(); got != "unknown error" {
		t.Errorf("reasonUnknown.message() = %q, want %q", got, "unknown error")
	}
}

func TestValidateNilOnSuccess(t *testing.T) {
	if err := Validate("(1+2)-3.5"); err != nil {
		t.Errorf("Validate returned %v, want nil", err)
	}
}

// Deep nesting must run in constant memory: an FSM only counts, a recursive
// parser would consume stack proportional to depth.
func TestDeepNesting(t *testing.T) {
	const depth = 1_000_000
	var b strings.Builder
	b.Grow(2*depth + 1)
	for range depth {
		b.WriteByte('(')
	}
	b.WriteByte('1')
	for range depth {
		b.WriteByte(')')
	}
	if !Valid(b.String()) {
		t.Error("deeply nested expression should be valid")
	}
	// One closing paren short: must fail, still in constant memory.
	if Valid(b.String()[:b.Len()-1]) {
		t.Error("unbalanced deep nesting should be invalid")
	}
}

func TestLongFlatExpression(t *testing.T) {
	var b strings.Builder
	b.WriteByte('1')
	for range 500_000 {
		b.WriteString("+1")
	}
	if !Valid(b.String()) {
		t.Error("long flat expression should be valid")
	}
}
