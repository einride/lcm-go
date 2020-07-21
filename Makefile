# all: run a complete build
.PHONY: all
all: \
	markdown-lint \
	go-lint \
	go-review \
	go-test \
	go-mod-tidy \
	git-verify-nodiff

include tools/git-verify-nodiff/rules.mk
include tools/golangci-lint/rules.mk
include tools/goreview/rules.mk
include tools/prettier/rules.mk

# go-mod-tidy: update Go module files
.PHONY: go-mod-tidy
go-mod-tidy:
	go mod tidy -v

# go-test: run Go test suite
.PHONY: go-test
go-test:
	go test -timeout 30s -race -cover ./...
