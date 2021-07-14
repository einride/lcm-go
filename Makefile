# all: run a complete build
.PHONY: all
all: \
	markdown-lint \
	go-mock-gen \
	go-lint \
	go-review \
	go-test \
	go-mod-tidy \
	git-verify-nodiff

include tools/git-verify-nodiff/rules.mk
include tools/golangci-lint/rules.mk
include tools/goreview/rules.mk
include tools/prettier/rules.mk
include tools/mockgen/rules.mk

# go-mod-tidy: update Go module files
.PHONY: go-mod-tidy
go-mod-tidy:
	go mod tidy -v

# go-test: run Go test suite
.PHONY: go-test
go-test:
	go test -timeout 30s -race -cover ./...

# go-mock-gen: generate Go mocks
.PHONY: go-mock-gen
go-mock-gen: $(mockgen) \
	test/mocks/player/service.go

test/mocks/player/service.go: $(mockgen)
	mkdir -p $(@D)
	$(mockgen) -package mockplayer -destination $@ \
	github.com/einride/lcm-go/pkg/player Transmitter
