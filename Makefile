# Build orchestrator for the Go game (module coredefense).
#
#   make             build both binaries into bin/
#   make prod        bin/coredefense          (production: no debug code)
#   make debug       bin/coredefense-debug    (-tags debug: tuning panel)
#   make test        go test ./...
#   make test-debug  go test -tags debug ./...
#   make vet         go vet ./...
#   make fmt         gofmt -l .               (report only)
#   make fmt-fix     gofmt -w .
#   make run         run the production binary
#   make run-debug   run the debug binary
#   make clean       remove bin/

GO  ?= go
BIN ?= bin

.PHONY: all prod debug test test-debug vet fmt fmt-fix run run-debug clean

all: prod debug

$(BIN):
	mkdir -p $(BIN)

SRCS := $(shell find . -name '*.go' -not -path './.git/*')

$(BIN)/coredefense: $(SRCS) go.mod go.sum | $(BIN)
	$(GO) build -o $@ .

$(BIN)/coredefense-debug: $(SRCS) go.mod go.sum | $(BIN)
	$(GO) build -tags debug -o $@ .

prod: $(BIN)/coredefense
debug: $(BIN)/coredefense-debug

test:
	$(GO) test ./...

test-debug:
	$(GO) test -tags debug ./...

vet:
	$(GO) vet ./...

fmt:
	@gofmt -l .

fmt-fix:
	gofmt -w .

run: prod
	$(BIN)/coredefense

run-debug: debug
	$(BIN)/coredefense-debug

clean:
	rm -rf $(BIN)
