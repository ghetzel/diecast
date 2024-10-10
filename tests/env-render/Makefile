TESTBIN ?= ../../bin/diecast-$(shell go env GOOS)-$(shell go env GOARCH)-nocgo
FILES   ?= $(wildcard *.yaml)

.EXPORT_ALL_VARIABLES:

all: $(FILES)

$(FILES):
	@$(TESTBIN) --page cool_beans=hello --render "$(@)" | shasum --quiet --algorithm 256 --check "$(@).sha256sum"
	@echo "$(@): OK"

.PHONY: $(FILES)