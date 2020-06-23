override SHELL := /usr/bin/env bash -euxo pipefail
override .DEFAULT_GOAL := all

all: force vet lint test

vet: force
	@go vet $(VETFLAGS) ./...

lint: force
	@go run golang.org/x/lint/golint -set_exit_status $(LINTFLAGS) ./...

test: force
	@rm -f coverage.txt
	@( \
	    TEMPFILE=$$(mktemp) && trap "rm \"$${TEMPFILE}\"" EXIT && \
	    go test -coverprofile="$${TEMPFILE}" -covermode=count $(TESTFLAGS) -coverpkg ./cmd/proxyz/... ./cmd/proxyz && \
	    cat "$${TEMPFILE}" >> coverage.txt \
	)
	@( \
	    TEMPFILE=$$(mktemp) && trap "rm \"$${TEMPFILE}\"" EXIT && \
	    go test -coverprofile="$${TEMPFILE}" -covermode=count $(TESTFLAGS) . && \
	    cat "$${TEMPFILE}" >> coverage.txt \
	)

.PHONY: force
force:
