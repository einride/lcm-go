all: \
    mod-tidy \
    go-test

export GO111MODULE = on

.PHONY: build
build:
	@git submodule update --init --recursive $@

include build/rules.mk
build/rules.mk: build
	@# Included in submodule: build

.PHONY: mod-tidy
mod-tidy:
	go mod tidy

.PHONY: go-test
go-test:
	go test -race -cover ./...
