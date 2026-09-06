// Package mathexpr validates basic arithmetic expressions.
//
// Accepted grammar (see README for rationale behind each decision):
//
//	expr    := operand { ("+" | "-") operand }
//	operand := [ "-" ] ( number | "(" expr ")" )   // unary minus only at
//	                                               // expression start or after "("
//	number  := digits [ "." digits ]               // "3." and ".5" are invalid
//
// ASCII spaces and tabs are allowed between tokens, never inside a number.
// Input is treated as bytes: any non-ASCII byte is invalid by construction.
//
// The validator is a single-pass finite state machine: O(n) time, O(1)
// memory, zero heap allocations on the accepting path.
package mathexpr

// state is the FSM state. The machine never allocates; all transitions are
// switches over the current byte.
type state uint8

const (
	// stateOperandU: expecting an operand; unary minus IS allowed
	// (expression start, or immediately after "("). If starting from the default value (0),
	// then something is wrong (cannot be reproduced through the public API)
	stateOperandU state = iota + 1
	// stateOperandN: expecting an operand; unary minus is NOT allowed
	// (immediately after a binary operator).
	stateOperandN
	// stateUnary: just consumed a unary minus; a digit or "(" must follow.
	stateUnary
	// stateInt: inside the integer part of a number.
	stateInt
	// stateFrac: just consumed "."; a digit must follow.
	stateFrac
	// stateFracDig: inside the fractional digits of a number.
	stateFracDig
	// stateAfterVal: a complete value has ended; expecting "+", "-", ")" or end.
	stateAfterVal
)

// failure reasons, used by Validate to build diagnostics.
type reason uint8

const (
	// reasonUnknown is the zero value; never produced by scan.
	reasonUnknown reason = iota
	reasonOK
	reasonEmpty
	reasonUnexpectedChar
	reasonUnbalancedParen
	reasonUnexpectedEnd
)

func (r reason) message() string {
	switch r {
	case reasonEmpty:
		return "empty expression"
	case reasonUnexpectedChar:
		return "unexpected character"
	case reasonUnbalancedParen:
		return "unbalanced parenthesis"
	case reasonUnexpectedEnd:
		return "unexpected end of expression"
	default:
		return "unknown error"
	}
}

// Valid reports whether expr is a syntactically valid arithmetic expression.
// It is the hot-path API: zero allocations, no error construction.
func Valid(expr string) bool {
	_, r := scan(expr)
	return r == reasonOK
}

// Validate checks expr and returns nil if it is valid, or a *SyntaxError
// describing the first failure. It allocates only on the failing path.
func Validate(expr string) error {
	pos, r := scan(expr)
	if r == reasonOK {
		return nil
	}
	return &SyntaxError{Pos: pos, Reason: r.message()}
}

// scan runs the FSM over expr and returns the byte offset and reason of the
// first failure, or (len(expr), reasonOK) on success.
func scan(expr string) (int, reason) {
	if len(expr) == 0 {
		return 0, reasonEmpty
	}

	st := stateOperandU
	var depth int // open-parenthesis depth; must end at zero

	for i := range len(expr) {
		c := expr[i]
		switch st {
		case stateOperandU:
			switch {
			case c == ' ' || c == '\t':
				// skip
			case c == '-':
				st = stateUnary
			case c == '(':
				depth++
				// stay in stateOperandU: unary minus allowed after "("
			case isDigit(c):
				st = stateInt
			case c == ')' && depth == 0:
				// a stray ")" where an operand is expected is a balance
				// problem, not a character problem - report it as such
				return i, reasonUnbalancedParen
			default:
				return i, reasonUnexpectedChar
			}

		case stateOperandN:
			switch {
			case c == ' ' || c == '\t':
				// skip
			case c == '(':
				depth++
				st = stateOperandU
			case isDigit(c):
				st = stateInt
			case c == ')' && depth == 0:
				return i, reasonUnbalancedParen
			default:
				// note: '-' lands here too - "1+-2" is rejected by design
				return i, reasonUnexpectedChar
			}

		case stateUnary:
			switch {
			case c == '(':
				depth++
				st = stateOperandU
			case isDigit(c):
				st = stateInt
			default:
				// space after unary minus ("- 5") is rejected by design
				return i, reasonUnexpectedChar
			}

		case stateInt:
			switch {
			case isDigit(c):
				// stay
			case c == '.':
				st = stateFrac
			case c == '+' || c == '-':
				st = stateOperandN
			case c == ')':
				if depth == 0 {
					return i, reasonUnbalancedParen
				}
				depth--
				st = stateAfterVal
			case c == ' ' || c == '\t':
				st = stateAfterVal
			default:
				return i, reasonUnexpectedChar
			}

		case stateFrac:
			if !isDigit(c) {
				return i, reasonUnexpectedChar
			}
			st = stateFracDig

		case stateFracDig:
			switch {
			case isDigit(c):
				// stay
			case c == '+' || c == '-':
				st = stateOperandN
			case c == ')':
				if depth == 0 {
					return i, reasonUnbalancedParen
				}
				depth--
				st = stateAfterVal
			case c == ' ' || c == '\t':
				st = stateAfterVal
			default:
				return i, reasonUnexpectedChar
			}

		case stateAfterVal:
			switch c {
			case ' ', '\t':
				// skip
			case '+', '-':
				st = stateOperandN
			case ')':
				if depth == 0 {
					return i, reasonUnbalancedParen
				}
				depth--
			default:
				return i, reasonUnexpectedChar
			}
		}
	}

	// End of input: only states representing a complete value may accept.
	switch st {
	case stateInt, stateFracDig, stateAfterVal:
		if depth != 0 {
			return len(expr), reasonUnbalancedParen
		}
		return len(expr), reasonOK
	default:
		return len(expr), reasonUnexpectedEnd
	}
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}
