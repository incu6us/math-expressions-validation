package mathexpr

import "testing"

// FuzzAgainstReference asserts the FSM and the oracle agree on every input.
func FuzzAgainstReference(f *testing.F) {
	seeds := []string{
		"", "1", "1+2", "-5", "(1+2)-3", "3.14", "()", "(", ")",
		"1+-2", "--5", "- 5", "3.", ".5", "1 + 2", "((1))", "1.2.3",
		"-(-5)", "1\n2", "0",
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		got, want := Valid(s), refValid(s)
		if got != want {
			t.Errorf("disagreement on %q: fsm=%t ref=%t", s, got, want)
		}
	})
}

// The oracle itself must pass the main table too - run it through the same
// cases to keep the two implementations honest against the written spec.
func TestReferenceAgainstTables(t *testing.T) {
	cases := map[string]bool{
		"1+2": true, "-5": true, "(-5)": true, "3-(-5)": true,
		"3+-5": false, "()": false, "3.": false, ".5": false,
		"1 . 5": false, "( 1 + 2 )": true,
	}
	for expr, want := range cases {
		if got := refValid(expr); got != want {
			t.Errorf("refValid(%q) = %t, want %t", expr, got, want)
		}
	}
}

// refValid is a deliberately naive recursive-descent validator implementing
// the same grammar as the FSM. It exists only as a differential oracle for
// fuzzing: two independent implementations agreeing on random inputs is far
// stronger evidence than either alone. It is test-only code; clarity over
// performance.
func refValid(s string) bool {
	p := &refParser{s: s}
	p.skipWS()
	if !p.expr(true) {
		return false
	}
	p.skipWS()
	return p.i == len(p.s)
}

type refParser struct {
	s string
	i int
}

func (p *refParser) peek() (byte, bool) {
	if p.i < len(p.s) {
		return p.s[p.i], true
	}
	return 0, false
}

func (p *refParser) skipWS() {
	for p.i < len(p.s) && (p.s[p.i] == ' ' || p.s[p.i] == '\t') {
		p.i++
	}
}

// expr := operand { (+|-) operand }
// unaryOK is true at expression start (incl. right after "(").
func (p *refParser) expr(unaryOK bool) bool {
	if !p.operand(unaryOK) {
		return false
	}
	for {
		p.skipWS()
		c, ok := p.peek()
		if !ok || (c != '+' && c != '-') {
			return true
		}
		p.i++ // consume operator
		p.skipWS()
		// after a binary operator unary minus is not allowed
		if !p.operand(false) {
			return false
		}
	}
}

// operand := ["-"] ( number | "(" expr ")" )
func (p *refParser) operand(unaryOK bool) bool {
	c, ok := p.peek()
	if !ok {
		return false
	}
	if c == '-' {
		if !unaryOK {
			return false
		}
		p.i++
		// no whitespace between unary minus and its operand
		c, ok = p.peek()
		if !ok {
			return false
		}
		if c == '(' {
			return p.paren()
		}
		return p.number()
	}
	if c == '(' {
		return p.paren()
	}
	return p.number()
}

func (p *refParser) paren() bool {
	p.i++ // consume "("
	p.skipWS()
	if !p.expr(true) {
		return false
	}
	p.skipWS()
	c, ok := p.peek()
	if !ok || c != ')' {
		return false
	}
	p.i++
	return true
}

// number := digits ["." digits]
func (p *refParser) number() bool {
	start := p.i
	for p.i < len(p.s) && isDigit(p.s[p.i]) {
		p.i++
	}
	if p.i == start {
		return false
	}
	if p.i < len(p.s) && p.s[p.i] == '.' {
		p.i++
		fracStart := p.i
		for p.i < len(p.s) && isDigit(p.s[p.i]) {
			p.i++
		}
		if p.i == fracStart {
			return false
		}
	}
	return true
}
