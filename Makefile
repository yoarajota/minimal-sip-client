.PHONY: quality test

# Single entry point: every declared check, exits non-zero on violation.
# Extended with the per-weakness tools at P3 (see .sota/quality-gates.yaml iso5055.weaknesses).
quality:
	@test -z "$$(gofmt -l $$(find . -name '*.go' -not -path '*/vendor/*'))" \
	  || { echo "gofmt needed on the files listed above"; exit 1; }
	go vet ./...
	go build -o /dev/null .
	cd poc && go vet ./... && go build -o /dev/null .

test:
	go test ./... -count=1
