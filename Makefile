.PHONY: test deps
.EXPORT_ALL_VARIABLES:

GO111MODULE ?= on
LOCALS      := $(shell find . -type f -name '*.go')

all: deps test build

deps:
	@go list github.com/mjibson/esc || go get github.com/mjibson/esc/...
	go generate -x ./...
	gofmt -w $(LOCALS)
	go vet $(PKGS)
	go get ./...

test:
	go test ./...

build:
	test -d diecast && go build -i -o bin/diecast diecast/main.go
	test -d diecast/funcdoc && go build -i -o bin/funcdoc diecast/funcdoc/main.go
	./bin/funcdoc > FUNCTIONS.md

package:
	-rm -rf pkg
	mkdir -p pkg/usr/bin
	cp bin/diecast pkg/usr/bin/diecast
	fpm \
		--input-type  dir \
		--output-type deb \
		--deb-user    root \
		--deb-group   root \
		--name        diecast \
		--version     `./pkg/usr/bin/diecast -v | cut -d' ' -f3` \
		-C            pkg
