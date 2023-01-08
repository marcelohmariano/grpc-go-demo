SHELL := /bin/sh
MAKEFLAGS += Rrs

.ONESHELL:
.SUFFIXES:

BIN := $(PWD)/.bin
export PATH := $(BIN):$(PATH)

GO_BUILD := go build -ldflags='-s -w' -o $(BIN)/
GO_INSTALL := $(GO_BUILD) -modfile=tools/go.mod

BINS = $(addprefix $(BIN)/,$(notdir $(shell find cmd -type d -mindepth 1 -maxdepth 1)))
BUF_MODS = $(dir $(shell find . -name buf.yaml -type f))

BUF := $(BIN)/buf
GOFMT := $(BIN)/goimports-reviser
GOLINT := $(BIN)/golangci-lint

.PHONY: all
all: gen bins

.PHONY: gen
gen: $(BUF)
	rm -rf gen && $(BUF) generate

.PHONY: bins
bins: $(BINS)

.PHONY: clean
clean:
	rm -f $(BINS)

.PHONY: fmt
fmt: $(BUF) $(GOFMT)
	$(BUF) format -w
	$(GOFMT) -rm-unused -set-alias -format ./... 2>/dev/null || true

.PHONY: lint
lint: $(BUF) $(GOLINT)
	$(BUF) lint
	$(GOLINT) run ./...

.PHONY: tidy
tidy: go-mod-tidy buf-mod-tidy

.PHONY: go-mod-tidy
go-mod-tidy:
	go mod tidy && cd tools && go mod tidy

.PHONY: buf-mod-tidy
buf-mod-tidy:
	for mod in $(BUF_MODS); do \
		cd "$$mod" && $(BUF) mod update && $(BUF) mod prune; \
	done

$(BIN)/%:
	$(GO_BUILD) ./cmd/$*

$(BUF):
	$(GO_INSTALL) 'github.com/bufbuild/buf/cmd/buf'

$(GOFMT):
	$(GO_INSTALL) 'github.com/incu6us/goimports-reviser/v3'

$(GOLINT):
	$(GO_INSTALL) 'github.com/golangci/golangci-lint/cmd/golangci-lint'
