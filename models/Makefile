.PHONY: test test-coverage

# Run unit tests
test:
	@go test ./...

# Run tests with race detector and coverage, then enforce the 85% gate on
# every non-test function in this module: the coverage report keys each
# function by file path, every function in the module lives under
# .../knot/models/, and Go's coverage report never lists test functions,
# so gating on that path covers all of them. Exits non-zero if any
# function is below 85.0%, or if the grep pattern matches nothing at all
# (e.g. the module was renamed) so the gate can never pass vacuously.
test-coverage:
	@go test ./... -race -covermode=atomic -coverprofile=coverage.out -coverpkg=./...
	@func_report=$$(go tool cover -func=coverage.out); \
	printf '%s\n' "$$func_report"; \
	gated=$$(printf '%s\n' "$$func_report" | grep -E 'knot/models'); \
	if [ -z "$$gated" ]; then \
		echo "COVERAGE GATE FAIL: no functions matched the module path pattern (renamed module?)"; \
		exit 1; \
	fi; \
	printf '%s\n' "$$gated" | awk '{gsub(/%/,"",$$NF); if ($$NF+0 < 85.0) { print "COVERAGE GATE FAIL:", $$1, $$NF"%"; f=1 }} END { exit f }'
