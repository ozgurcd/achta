SHELL := /bin/sh

.PHONY: help build test fmt-check verify install-tools

help:
	@printf '%s\n' 'Targets:' '  build          build ./cmd/achta' '  test           run unit tests' '  fmt-check      fail on unformatted Go files' '  verify         run the complete local verification gate' '  install-tools  install pinned verification tools'

build:
	go build ./...

test:
	go test ./... -count=1 -timeout=120s

fmt-check:
	@test -z "$$(gofmt -l cmd internal)" || { gofmt -l cmd internal; exit 1; }

verify: fmt-check
	go version
	go build ./...
	go test ./... -count=1 -timeout=120s
	go vet ./...
	STATICCHECK_CACHE=$(CURDIR)/.cache/staticcheck staticcheck ./...
	govulncheck ./...
	go mod tidy -diff
	rulefloor check --repo .

install-tools:
	go install honnef.co/go/tools/cmd/staticcheck@v0.8.0
	go install golang.org/x/vuln/cmd/govulncheck@v1.3.0
	go install github.com/ozgurcd/rulefloor@v0.9.1
