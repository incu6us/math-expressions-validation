package mathexpr

import "strconv"

// SyntaxError describes the first syntax failure in an expression.
type SyntaxError struct {
	Pos    int    // byte offset of the failure (== len(expr) for truncated input)
	Reason string // human-readable reason
}

func (e *SyntaxError) Error() string {
	return "mathexpr: " + e.Reason + " at position " + strconv.Itoa(e.Pos)
}
