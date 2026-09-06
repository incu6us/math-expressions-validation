# math-expressions-validation

[![CI](https://github.com/incu6us/math-expressions-validation/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/incu6us/math-expressions-validation/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/incu6us/math-expressions-validation/branch/main/graph/badge.svg)](https://codecov.io/gh/incu6us/math-expressions-validation)

A validator for basic arithmetic expressions in Go: decimal numbers
(positive and negative), `+`, `-`, and parentheses.

It answers one question - *is this expression syntactically valid?* - and is
built around the task's stated focus: **performance**. The core is a
single-pass finite state machine: O(n) time, O(1) memory, zero heap
allocations on the accepting path.

## API

```go
import mathexpr "github.com/incu6us/math-expressions-validation"
```

```go
// Hot path: zero allocations, no error construction.
func Valid(expr string) bool

// Diagnostic path: nil on success, or *SyntaxError{Pos, Reason} describing
// the first failure. Allocates only when the input is invalid.
func Validate(expr string) error
```

Two tiers on purpose: a caller filtering high volumes of input wants a
branch, not an error value; a caller surfacing messages to a human wants a
position. Both share the same scanner, so they cannot drift apart.

## Grammar

```
expr    := operand { ("+" | "-") operand }
operand := [ "-" ] ( number | "(" expr ")" )
number  := digits [ "." digits ]
```

The task intentionally leaves edge semantics open. Every decision below is a
choice, made explicit; each is a one-line change in the FSM if the reviewer
prefers different semantics.

| Case | Verdict | Rationale |
|---|---|---|
| `3.` / `.5` | invalid | digits required on both sides of the dot; strictest rule, no ambiguity |
| `-5`, `(-5)`, `-(1+2)` | valid | unary minus at expression start or after `(` |
| `3+-5`, `--5` | invalid | unary minus after a binary operator is rejected; write `3+(-5)`. The alternative reading - `+-5` as "plus a negative number" - was considered and rejected: chained sign characters are a classic typo (`3+-5` for `3+5` or `3-5`), and a validator's job is to catch those. Allowing it is a one-line FSM change (`stateOperandN`: `'-'` -> `stateUnary`) |
| `- 5` | invalid | unary minus must be attached to its operand |
| `+5` | invalid | no unary plus |
| `()` | invalid | empty groups are meaningless |
| `007` | valid | leading zeros are a style question, not a syntax one |
| `1 + 2` | valid | ASCII space/tab allowed between tokens |
| `1 . 5`, `1 2` | invalid | no whitespace inside numbers; no implicit operators |
| newline, NBSP, non-ASCII | invalid | input is processed as bytes; the grammar is pure ASCII, so any non-ASCII byte fails by construction |

## Design

**Why not regex.** Balanced parentheses of arbitrary depth are not a regular
language - no regex can validate them. (And backtracking engines invite
pathological inputs, the opposite of the performance goal.)

**Why not a tokenizer + parser + AST.** The task asks for validation, not
evaluation. A parse tree is structure built only to be discarded: it costs
allocations, and recursion makes stack usage proportional to nesting depth.

**Why an FSM.** A handful of states, one
integer for parenthesis depth, one switch
per input byte. Constant memory regardless of input: the test suite includes
a **1,000,000-deep** nested expression, which validates in ~2 ms; a
recursive parser would overflow the stack far earlier.

The only subtlety worth calling out: two "expecting operand" states -
one where unary minus is legal (start, after `(`) and one where it is not
(after a binary operator). That distinction is the whole unary-minus grammar.

## Performance

`go test -bench=. -benchtime=1s` (Go 1.27, darwin/arm64, Apple M4 Pro -
relative numbers are the point):

| input | ns/op | throughput | allocs/op |
|---|---|---|---|
| `1+2` | 4.2 | 710 MB/s | 0 |
| `(12.5-3)+(4-(-2.75))` | 20.0 | 998 MB/s | 0 |
| flat, 20 KB | 17,323 | 1155 MB/s | 0 |
| nested, 10k deep | 20,262 | 987 MB/s | 0 |

The module declares `go 1.24` - the true minimum (`testing.B.Loop`); the
numbers above were measured on Go 1.27. CI tests both the minimum and the
current toolchain.

Benchmarks use `testing.B.Loop` (Go 1.24+), which prevents the compiler from
optimizing away the benchmarked call - the numbers are slightly higher than a
classic `b.N` loop would show, and slightly more honest for the same reason.

The zero-allocation claim is also visible in the compiler's escape
analysis (`go build -gcflags=-m`, trimmed):

```
can inline Valid
expr does not escape                 (Valid, Validate and scan alike)
&SyntaxError{...} escapes to heap    // failing path only
```

(line:column prefixes stripped - they shift with every edit; the facts don't)

Nothing escapes on the accepting path; the single heap allocation in the
package is the error value `Validate` builds for invalid input - and
`TestZeroAllocs` enforces this stays true.

Against a straightforward recursive-descent reference implementation of the
same grammar (included in the test package as the fuzzing oracle): the FSM
is ~1.5-2x faster on typical input and ~7x on deep nesting, with no stack
growth.

## Testing

- **Table-driven tests** grouped by grammar area (numbers, unary minus,
  operators, parentheses, whitespace, encoding). The tables are the
  executable form of the grammar table above.
- **Differential fuzzing**: a deliberately naive recursive-descent validator
  (test-only) implements the same grammar independently; the fuzzer asserts
  both implementations agree on every input. 18M+ executions clean.
  Two independent implementations agreeing is much stronger evidence than
  either implementation's own tests.
- **Degenerate-input tests**: 1M-deep nesting (valid and off-by-one
  invalid), 1 MB flat expression, empty and whitespace-only input.
- Position reporting of `Validate` is tested separately.
- **Zero-allocation guarantee** is enforced by `TestZeroAllocs`
  (`testing.AllocsPerRun`) - a regression that introduces an allocation
  fails `make test` and CI, not just a benchmark table.

The library sits at 100% statement coverage, with the zero-allocation
guarantee enforced as a test rather than observed in benchmarks.

Run everything:

```
make install   # installs golangci-lint if missing (the only tool; module has zero deps)
make check     # build + lint + test + vuln in one go
make build
make test
make lint      # golangci-lint, incl. gofumpt formatting check
make vuln      # govulncheck against the Go vulnerability database (also in CI)
make run       # runs the demo program in ./example
make bench
make fuzz      # 30s differential fuzzing against the reference oracle
```

CI (`.github/workflows/ci.yml`) runs build, lint, test, bench and fuzz via
the same Makefile targets. To validate it locally before pushing:
`brew install act && act push` (flags for Apple Silicon are in `.actrc`).

## Out of scope

- **Evaluation** - the task asks for a validator; no value is computed.
- **`*`, `/`, precedence** - not in the task. Validation-wise they would be
  additional operator bytes in two states (precedence only matters for
  evaluation, not validity).
- **Unicode digits / localized formats** - grammar is ASCII by decision.
- **Overflow / numeric range checks** - syntax validity is independent of
  representability; `9999999999999999999999` is syntactically fine.
