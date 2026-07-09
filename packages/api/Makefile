.PHONY: run test test-verbose check-env

run: check-env
	go run ./src

test:
	go test ./...

test-verbose:
	go test -v ./...

check-env:
	@test -n "$$OPENROUTER_API_KEY" \
		|| { echo "OPENROUTER_API_KEY is not set"; exit 1; }
	@test -n "$$OPENROUTER_MODEL" \
		|| { echo "OPENROUTER_MODEL is not set"; exit 1; }
