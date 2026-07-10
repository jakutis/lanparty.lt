.PHONY: run test test-verbose check-env curl

curl:
	curl -XPOST localhost:$(or $(PORT),8080)/v1/representation \
		-d '{"type":"html","spec":"a hello world web page"}'

run: check-env
	go run ./src

test:
	go test ./...
	go test -C ../verification ./...

test-verbose:
	go test -v ./...
	go test -C ../verification -v ./...

check-env:
	@test -n "$$OPENROUTER_API_KEY" \
		|| { echo "OPENROUTER_API_KEY is not set"; exit 1; }
	@test -n "$$OPENROUTER_MODEL" \
		|| { echo "OPENROUTER_MODEL is not set"; exit 1; }
