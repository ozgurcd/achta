SHELL := /bin/sh

.PHONY: help build test test-fuzz fmt-check wiki-check replacement-check release-notes-check verify rulefloor-static install-tools

FUZZTIME ?= 5s
RELEASE_NOTES_OUTPUT ?= /dev/null

help:
	@printf '%s\n' 'Targets:' '  build               build ./cmd/achta' '  test                run unit tests' '  test-fuzz           fuzz every narrow parser for FUZZTIME each' '  fmt-check           fail on unformatted Go files' '  wiki-check          validate the co-versioned repository wiki' '  replacement-check   require cited replay for named script replacement claims' '  release-notes-check run the release extractor for the source version' '  verify              run the complete local verification gate' '  rulefloor-static    validate the ledger without executing bindings' '  install-tools       install pinned verification tools'

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

wiki-check:
	go run ./cmd/achta --workspace . wiki check --json

replacement-check:
	go run ./cmd/achta --workspace . replacement check --file replacement-claims.json --claim-status replaces --claim-status retired --json

release-notes-check:
	go run ./cmd/release-notes RELEASE_NOTES.md > "$(RELEASE_NOTES_OUTPUT)"

verify: fmt-check release-notes-check
	go version
	go build ./...
	go test ./... -count=1 -timeout=120s
	go vet ./...
	STATICCHECK_CACHE=$(CURDIR)/.cache/staticcheck staticcheck ./...
	govulncheck ./...
	go mod tidy -diff
	gograph build . --precise
	$(MAKE) wiki-check
	$(MAKE) replacement-check
	rulefloor check --repo . --run-profile unit --timings

rulefloor-static:
	rulefloor check --repo .

install-tools:
	go install honnef.co/go/tools/cmd/staticcheck@v0.8.0
	go install golang.org/x/vuln/cmd/govulncheck@v1.3.0
	go install github.com/ozgurcd/rulefloor@v0.9.1
	go install github.com/ozgurcd/gograph/cmd/gograph@v1.6.10
