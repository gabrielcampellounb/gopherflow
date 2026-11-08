SHELL := /bin/sh

GO := go
TOOLS_DIR := .tools/bin
STATICCHECK := $(TOOLS_DIR)/staticcheck

.PHONY: help tools build run test race fmt fmt-check vet staticcheck lint coverage coverage-html check clean

help:
	@printf '%s\n' \
		'help          Show available targets' \
		'tools         Install local analysis tools' \
		'build         Build the CLI' \
		'run           Run the CLI' \
		'test          Run unit tests' \
		'race          Run tests with the race detector' \
		'fmt           Format Go files' \
		'fmt-check     Check Go formatting' \
		'vet           Run go vet' \
		'staticcheck   Run staticcheck' \
		'lint          Run formatting and static analysis' \
		'coverage      Generate coverage summary' \
		'coverage-html Generate HTML coverage report' \
		'check         Run lint, tests, and race tests' \
		'clean         Remove generated artifacts'

tools:
	mkdir -p $(TOOLS_DIR)
	GOBIN="$(PWD)/$(TOOLS_DIR)" $(GO) install honnef.co/go/tools/cmd/staticcheck@latest

build:
	mkdir -p bin
	$(GO) build -o bin/gopherflow ./cmd/gopherflow

run:
	$(GO) run ./cmd/gopherflow

test:
	$(GO) test ./...

race:
	$(GO) test -race ./...

fmt:
	find . -type f -name '*.go' -not -path './.tools/*' -print0 | xargs -0 -r gofmt -w

fmt-check:
	@files=$$(find . -type f -name '*.go' -not -path './.tools/*' -print); \
	if [ -n "$$files" ] && [ -n "$$(gofmt -l $$files)" ]; then \
		gofmt -l $$files; exit 1; \
	fi

vet:
	$(GO) vet ./...

staticcheck: tools
	$(STATICCHECK) ./...

lint: fmt-check vet staticcheck

coverage:
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -func=coverage.out

coverage-html: coverage
	$(GO) tool cover -html=coverage.out -o coverage.html

check: lint test race

clean:
	rm -rf bin coverage.out coverage.html
