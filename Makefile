all: mod-tidy go-test

include build/rules.mk
build/rules.mk:
	git submodule update --init --recursive

.PHONY: mod-tidy
	go mod tidy

.PHONY: go-test
go-test:
	go test -race -cover ./...
