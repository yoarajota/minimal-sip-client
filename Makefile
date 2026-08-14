TOOLS := .tools
export GOBIN := $(CURDIR)/$(TOOLS)

.PHONY: quality test tools suite

# Project-local quality toolchain (rule R10 — nothing installed globally).
# Versions are pinned; every tool is built into .tools/ by `go install`.
tools:
	mkdir -p $(TOOLS)
	go install github.com/fzipp/gocyclo/cmd/gocyclo@v0.6.0
	go install github.com/kisielk/errcheck@v1.7.0
	go install golang.org/x/tools/cmd/deadcode@v0.24.0
	go install github.com/zricethezav/gitleaks/v8@v8.21.2
	go install honnef.co/go/tools/cmd/staticcheck@2025.1.1
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.60.3

# Single entry point: every declared ISO/IEC 5055 check, exits non-zero on
# violation. The poc/ module (throwaway evidence) is vet/build-checked but is
# not subject to the full gate.
quality: tools
	@test -z "$$(gofmt -l $$(find . -name '*.go' -not -path './vendor/*' -not -path './.tools/*' -not -path './.git/*'))" \
	  || { echo "gofmt needed on the files listed above"; exit 1; }
	go vet ./...
	go build -o /dev/null .
	cd poc && go vet ./... && go build -o /dev/null .
	$(TOOLS)/staticcheck ./...
	$(TOOLS)/errcheck -ignoretests -ignore 'net:.*,github.com/yoarajota/minimal-sip-client/internal/sip:Close' ./...
	$(TOOLS)/gocyclo -over 15 -ignore 'poc/' .
	$(TOOLS)/deadcode -test ./...
	$(TOOLS)/gitleaks detect --no-banner
	$(TOOLS)/golangci-lint run

test:
	go test ./... -count=1

# Integration suite against the real PBX (docker compose; see run-suite.sh).
suite:
	./run-suite.sh
