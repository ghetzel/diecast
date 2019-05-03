BIN ?= diecast-$(shell go env GOOS)-$(shell go env GOARCH)

all:
	../bin/$(BIN) -a 127.0.0.1:0

local:
	cd src && ../../bin/$(BIN) -p local=true -m /diecast/src:.
