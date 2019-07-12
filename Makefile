.PHONY: all
all: \
	markdown-lint \
	go-lint \
	go-test \
	go-mod-tidy

.PHONY: clean
clean:
	rm -rf build test/mocks

export GO111MODULE := on

.PHONY: build
build:
	@git submodule update --init --recursive $@

include build/rules.mk
build/rules.mk: build
	@# Included in submodule: build

.PHONY: markdown-lint
markdown-lint: $(PRETTIER)
	$(PRETTIER) --check **/*.md --parser markdown

.PHONY: go-mod-tidy
go-mod-tidy:
	go mod tidy -v

.PHONY: go-lint
go-lint: $(GOLANGCI_LINT) mocks
	$(GOLANGCI_LINT) run --enable-all

.PHONY: go-test
go-test: mocks
	go test -race -cover ./...

.PHONY: mocks
mocks: test/mocks/lcm.go

test/mocks/lcm.go: receiver.go transmitter.go $(GOBIN)
	$(GOBIN) -m -run github.com/golang/mock/mockgen -destination $@ \
		github.com/einride/lcm-go UDPReader,UDPWriter
