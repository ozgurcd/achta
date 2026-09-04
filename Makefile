SHELL := /bin/sh

.PHONY: help build test test-fuzz fmt-check verify rulefloor-static install-tools

FUZZTIME ?= 5s

help:
	@printf '%s\n' 'Targets:' '  build             build ./cmd/achta' '  test              run unit tests' '  test-fuzz         fuzz every narrow parser for FUZZTIME each' '  fmt-check         fail on unformatted Go files' '  verify            run the complete local verification gate' '  rulefloor-static  validate the ledger without executing bindings' '  install-tools     install pinned verification tools'

build:
	go build ./...

test:
	go test ./... -count=1 -timeout=120s

test-fuzz:
	go test ./internal/wiki -run '^$$' -fuzz '^FuzzRenderPin$$' -fuzztime=$(FUZZTIME)
	go test ./internal/wiki -run '^$$' -fuzz '^FuzzDerivedBlockDelimiters$$' -fuzztime=$(FUZZTIME)
	go test ./internal/decision -run '^$$' -fuzz '^FuzzAddDecisionHeading$$' -fuzztime=$(FUZZTIME)
	go test ./internal/witness -run '^$$' -fuzz '^FuzzParse$$' -fuzztime=$(FUZZTIME)
	go test ./internal/amendments -run '^$$' -fuzz '^FuzzParse$$' -fuzztime=$(FUZZTIME)

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
	rulefloor check --repo . --run-profile unit --timings

rulefloor-static:
	rulefloor check --repo .

install-tools:
	go install honnef.co/go/tools/cmd/staticcheck@v0.8.0
	go install golang.org/x/vuln/cmd/govulncheck@v1.3.0
	go install github.com/ozgurcd/rulefloor@v0.9.1
	go install github.com/ozgurcd/gograph/cmd/gograph@v1.6.10
