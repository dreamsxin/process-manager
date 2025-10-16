.PHONY: build test clean examples

build:
	@echo "Building process manager..."
	@go build ./...

test:
	@echo "Running tests..."
	@go test ./tests/...

examples:
	@echo "Building examples..."
	@go build -o examples/basic/basic examples/basic/main.go

clean:
	@echo "Cleaning..."
	@rm -f examples/*/*

lint:
	@golangci-lint run

release:
	@goreleaser release --rm-dist

.DEFAULT_GOAL := build