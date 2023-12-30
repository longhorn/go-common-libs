TARGETS := $(shell ls dapper)

.DEFAULT_GOAL := ci

.PHONY: $(TARGETS)

.dapper:
	@echo Downloading dapper
	@curl -sSfL https://releases.rancher.com/dapper/latest/dapper-`uname -s`-`uname -m` > .dapper
	@chmod +x .dapper
	@./.dapper -v

$(TARGETS): .dapper
	./.dapper $@

deps: .dapper
	./.dapper -d -m bind go mod vendor
	./.dapper -d -m bind chown -R $$(id -u) vendor go.mod go.sum
