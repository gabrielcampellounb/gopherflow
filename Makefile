SHELL := /bin/sh

GO := go
TOOLS_DIR := .tools/bin
STATICCHECK := $(TOOLS_DIR)/staticcheck

.PHONY: help tools build run test race fmt fmt-check vet staticcheck lint coverage coverage-html check clean

help:
	@printf '%s\n' \
		'help          Lista os alvos disponíveis' \
		'tools         Instala as ferramentas de análise locais' \
		'build         Compila o programa de linha de comando' \
		'run           Executa o programa de linha de comando' \
		'test          Executa os testes unitários' \
		'race          Executa os testes com o detector de corrida' \
		'fmt           Formata os arquivos Go' \
		'fmt-check     Verifica a formatação dos arquivos Go' \
		'vet           Executa o go vet' \
		'staticcheck   Executa o staticcheck' \
		'lint          Verifica a formatação e a análise estática' \
		'coverage      Gera o resumo de cobertura' \
		'coverage-html Gera o relatório HTML de cobertura' \
		'check         Executa todas as verificações' \
		'clean         Remove os artefatos gerados'

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
