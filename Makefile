# Makefile for foundation-go testing

.PHONY: help test test-all test-verbose test-core test-adapters test-errors test-log test-di test-assert test-tier1 coverage coverage-html coverage-func clean

# Default target
help:
	@echo "Foundation-go Test Makefile"
	@echo ""
	@echo "Available targets:"
	@echo "  make test              - Run all tests"
	@echo "  make test-verbose      - Run all tests with verbose output"
	@echo "  make test-core         - Run core package tests only"
	@echo "  make test-adapters     - Run adapter tests only"
	@echo "  make test-errors       - Run errors package tests"
	@echo "  make test-log          - Run log package tests"
	@echo "  make test-di           - Run DI (core/di.go) tests"
	@echo "  make test-assert       - Run test/assert tests"
	@echo "  make test-tier1        - Run Tier 1 adapter tests only"
	@echo "  make coverage          - Generate coverage report"
	@echo "  make coverage-html     - Generate HTML coverage report"
	@echo "  make coverage-func     - Show function-level coverage"
	@echo "  make build             - Build all packages"
	@echo "  make clean             - Clean test cache and coverage files"
	@echo ""

# Run all tests
test:
	@echo "Running all tests..."
	GO_ENV=test go test ./... -cover

# Run all tests with verbose output
test-verbose:
	@echo "Running all tests (verbose)..."
	GO_ENV=test go test ./... -v

# Alias for test-verbose
test-all: test-verbose

# Run core package tests
test-core:
	@echo "Running core package tests..."
	GO_ENV=test go test ./core/... -v -cover

# Run adapter tests
test-adapters:
	@echo "Running adapter tests..."
	GO_ENV=test go test ./adapters/... -v -cover

# Run errors package tests
test-errors:
	@echo "Running errors package tests..."
	GO_ENV=test go test ./errors/... -v -cover

# Run log package tests
test-log:
	@echo "Running log package tests..."
	GO_ENV=test go test ./log/... -v -cover

# Run DI tests (core/di_test.go)
test-di:
	@echo "Running DI tests..."
	GO_ENV=test go test ./core/... -run TestDI -v

# Run test/assert tests
test-assert:
	@echo "Running assertion tests..."
	GO_ENV=test go test ./test/... -run TestAssertions -v

# Run Tier 1 adapter tests only
test-tier1:
	@echo "Running Tier 1 adapter tests..."
	@echo "  - secrets_faker_impl.go"
	@echo "  - cache_impl.go (InMemory)"
	@echo "  - idempotency_store.go"
	@echo "  - pubsub_impl.go (Fake)"
	@echo "  - em_impl.go"
	@echo "  - sentry_error_reporter.go"
	@echo ""
	GO_ENV=test go test ./adapters/... \
		-run "TestFakeSecret|TestNewCache|TestMustNewCache|TestInMemoryCache|TestIdempotency|TestNewPubSub|TestMustNewPubSub|TestFakePubSub|TestEntityManager|TestSentryError" \
		-v

# Generate coverage report
coverage:
	@echo "Generating coverage report..."
	GO_ENV=test go test ./... -coverprofile=coverage.out
	@echo ""
	@echo "Coverage summary:"
	@go tool cover -func=coverage.out | tail -1

# Generate HTML coverage report and open in browser
coverage-html:
	@echo "Generating HTML coverage report..."
	GO_ENV=test go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report saved to coverage.html"
	@echo "Opening in browser..."
	@command -v open >/dev/null 2>&1 && open coverage.html || \
	command -v xdg-open >/dev/null 2>&1 && xdg-open coverage.html || \
	echo "Please open coverage.html manually"

# Show function-level coverage
coverage-func:
	@echo "Generating function-level coverage..."
	GO_ENV=test go test ./... -coverprofile=coverage.out
	@echo ""
	go tool cover -func=coverage.out

# Show coverage for specific packages
coverage-core:
	@echo "Core package coverage:"
	GO_ENV=test go test ./core/... -coverprofile=coverage.out
	@go tool cover -func=coverage.out | grep -E "(core/|total:)"

coverage-adapters:
	@echo "Adapter package coverage:"
	GO_ENV=test go test ./adapters/... -coverprofile=coverage.out
	@go tool cover -func=coverage.out | grep -E "(adapters/|total:)"

coverage-errors:
	@echo "Errors package coverage:"
	GO_ENV=test go test ./errors/... -coverprofile=coverage.out
	@go tool cover -func=coverage.out

# Build all packages
build:
	@echo "Building all packages..."
	go build ./...
	@echo "Build successful!"

# Clean test cache and coverage files
clean:
	@echo "Cleaning test cache and coverage files..."
	go clean -testcache
	rm -f coverage.out coverage.html
	@echo "Clean complete!"

# Run tests with race detector
test-race:
	@echo "Running tests with race detector..."
	GO_ENV=test go test ./... -race

# Run tests with race detector (verbose)
test-race-verbose:
	@echo "Running tests with race detector (verbose)..."
	GO_ENV=test go test ./... -race -v

# Quick test (core + adapters only)
test-quick:
	@echo "Running quick tests (core + adapters)..."
	GO_ENV=test go test ./core/... ./adapters/... ./errors/... ./log/... -cover

# Test with timeout
test-timeout:
	@echo "Running tests with 30s timeout..."
	GO_ENV=test go test ./... -timeout 30s -cover

# Show test count
test-count:
	@echo "Counting tests..."
	@GO_ENV=test go test ./... -v 2>&1 | grep -c "^=== RUN" || echo "0"

# Run only fast tests (no integration)
test-unit:
	@echo "Running unit tests only..."
	GO_ENV=test go test ./... -short -v

# Continuous testing (watch mode)
# Requires: go install github.com/cespare/reflex@latest
watch:
	@echo "Starting watch mode (requires reflex)..."
	@command -v reflex >/dev/null 2>&1 || (echo "Error: reflex not installed. Run: go install github.com/cespare/reflex@latest" && exit 1)
	reflex -r '\.go$$' -s -- sh -c 'clear && GO_ENV=test go test ./... -cover'

# Benchmark tests
bench:
	@echo "Running benchmarks..."
	GO_ENV=test go test ./... -bench=. -benchmem

# List all test functions
list-tests:
	@echo "Listing all test functions..."
	@find . -name "*_test.go" -type f -exec grep -h "^func Test" {} \; | sed 's/func //' | sed 's/(.*$$//' | sort

# Show test statistics
stats:
	@echo "Test Statistics"
	@echo "==============="
	@echo ""
	@echo "Total test files:"
	@find . -name "*_test.go" -type f | wc -l
	@echo ""
	@echo "Total test functions:"
	@find . -name "*_test.go" -type f -exec grep -h "^func Test" {} \; | wc -l
	@echo ""
	@echo "Tests by package:"
	@echo "  - core/:"
	@find ./core -name "*_test.go" -type f 2>/dev/null -exec grep -h "^func Test" {} \; | wc -l || echo "    0"
	@echo "  - adapters/:"
	@find ./adapters -name "*_test.go" -type f 2>/dev/null -exec grep -h "^func Test" {} \; | wc -l || echo "    0"
	@echo "  - errors/:"
	@find ./errors -name "*_test.go" -type f 2>/dev/null -exec grep -h "^func Test" {} \; | wc -l || echo "    0"
	@echo "  - log/:"
	@find ./log -name "*_test.go" -type f 2>/dev/null -exec grep -h "^func Test" {} \; | wc -l || echo "    0"
	@echo "  - test/:"
	@find ./test -name "*_test.go" -type f 2>/dev/null -exec grep -h "^func Test" {} \; | wc -l || echo "    0"
	@echo ""

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Run linter (requires golangci-lint)
lint:
	@echo "Running linter..."
	@command -v golangci-lint >/dev/null 2>&1 || (echo "Error: golangci-lint not installed" && exit 1)
	golangci-lint run

# Run all quality checks
check: fmt lint test
	@echo ""
	@echo "All checks passed!"

# CI target (what CI should run)
ci: clean build test coverage
	@echo ""
	@echo "CI checks complete!"
