all: go-test

include build/rules.mk
build/rules.mk:
	git submodule update --init --recursive

.PHONY: dep-ensure
dep-ensure: $(DEP)
	$(DEP) ensure -v

.PHONY: go-test
go-test: dep-ensure
	go test -race -cover ./...
