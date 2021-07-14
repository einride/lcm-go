mockgen_cwd := $(abspath $(dir $(lastword $(MAKEFILE_LIST))))
mockgen_version := 1.6.0
mockgen := $(mockgen_cwd)/$(mockgen_version)/mockgen

mockgen_archive_url := https://github.com/golang/mock/releases/download/v$(mockgen_version)/mock_$(mockgen_version)_linux_amd64.tar.gz
$(mockgen):
	$(info [mock-gen] fetching $(mockgen_version) binary)
	@mkdir -p $(dir $@)
	@curl -sSL $(mockgen_archive_url) -o - | \
    		tar -xz --directory $(dir $@) --strip-components 1
	@chmod +x $@
	@touch $@
