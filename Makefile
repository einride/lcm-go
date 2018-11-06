all: test

include build/rules.mk
build/rules.mk:
	git submodule update --init

.PHONY: test
test: vendor
	go test -race -cover ./...
