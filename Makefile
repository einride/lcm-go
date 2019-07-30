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
.PHONY: go-lint
go-lint: $(GOLANGCI_LINT) go-mocks
	$(GOLANGCI_LINT) run --enable-all

# go-test: run Go test suite
.PHONY: go-test
go-test: go-mocks
	go test -race -cover ./...

# go-mocks: generate Go mocks
.PHONY: go-mocks
go-mocks: test/mocks/lcm.go

test/mocks/lcm.go: receiver.go transmitter.go $(GOBIN)
	$(GOBIN) -m -run github.com/golang/mock/mockgen -destination $@ \
		github.com/einride/lcm-go UDPReader,UDPWriter
