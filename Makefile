.PHONY: deps fmt build

all: fmt deps build

deps:
	@go list golang.org/x/tools/cmd/goimports || go get golang.org/x/tools/cmd/goimports
	go generate -x
	go get .

fmt:
	goimports -w .
	go vet .

test: fmt
	go test .

build:
	go build -o bin/`basename ${PWD}` cli/*.go

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
