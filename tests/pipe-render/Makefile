TESTBIN ?= ../../bin/diecast-$(shell go env GOOS)-$(shell go env GOARCH)-nocgo
.EXPORT_ALL_VARIABLES:

all:
	@cat input.json | $(TESTBIN) --render test.tmpl | shasum --quiet --algorithm 256 --check "output.sha256sum"
	@echo "pipe-render: OK"

.PHONY: all