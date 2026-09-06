.PHONY: check
check: build lint test vuln

.PHONY: install
install:
	@command -v golangci-lint >/dev/null || go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
	@command -v govulncheck >/dev/null || go install golang.org/x/vuln/cmd/govulncheck@latest

.PHONY: build
build:
	go build ./...

.PHONY: run
run:
	go run ./example

.PHONY: test
test:
	go test ./...

.PHONY: cover
cover:
	go test -coverprofile=coverage.out ./...

.PHONY: lint
lint:
	golangci-lint run

.PHONY: vuln
vuln:
	govulncheck ./...

.PHONY: bench
bench:
	go test -bench=. -run '^$$' -benchmem

.PHONY: fuzz
fuzz:
	go test -fuzz='^Fuzz' -fuzztime=30s
