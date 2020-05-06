# all: run a complete build
.PHONY: all
all: \
	markdown-lint \
	go-lint \
	go-test \
	go-mod-tidy

# clean: remove generated build files
.PHONY: clean
clean:
	rm -rf build test/mocks

export GO111MODULE := on

.PHONY: build
build:
	@git submodule update --init --recursive $@

include build/rules.mk
build/rules.mk: build
	@# included in submodule: build

# markdown-lint: lint Markdown files
.PHONY: markdown-lint
markdown-lint: $(PRETTIER)
	$(PRETTIER) --check **/*.md --parser markdown

# go-mod-tidy: update Go module files
.PHONY: go-mod-tidy
go-mod-tidy:
	go mod tidy -v

# go-lint: lint Go code
# funlen: tests with many testcases become too long, but should not be split.
# unused: buggy with GolangCI-Lint 1.18.0
# godox: we keep todos in the history
# wsl: doesn't match Einrides style guides
# gomnd: Normally good, but not very helpful in this lowlevel package
# staticcheck: TODO: enable when migrated away from deprecated protobuf APIs
# testpackage: complains about using external package name for tests
.PHONY: go-lint
go-lint: $(GOLANGCI_LINT)
	$(GOLANGCI_LINT) run --enable-all --disable funlen,godox,wsl,gomnd,staticcheck,testpackage

# go-test: run Go test suite
.PHONY: go-test
go-test:
	go test -race -cover ./...
