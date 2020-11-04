TESTBIN ?= ../../bin/diecast-linux-amd64-nocgo
.EXPORT_ALL_VARIABLES:

all:
	@cat input.json | $(TESTBIN) --render test.tmpl | shasum --quiet --algorithm 256 --check "output.sha256sum"
	@echo "pipe-render: OK"

.PHONY: all