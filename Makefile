.PHONY: test deps

PKGS=`go list ./... | grep -v /vendor/`
LOCALS=`find . -type f -name '*.go' -not -path "./vendor/*"`

all: deps fmt build

deps:
	@which dep || go get -u github.com/golang/dep/cmd/dep
	@go list github.com/mjibson/esc || go get github.com/mjibson/esc/...
	@go list golang.org/x/tools/cmd/goimports || go get golang.org/x/tools/cmd/goimports
	go generate -x
	dep ensure

clean-bundle:
	-rm -rf public

clean:
	-rm -rf bin

fmt:
	goimports -w $(LOCALS)
	go vet $(PKGS)

test:
	go test --tags json1 $(PKGS)

build: deps fmt
	test -d cli && go build --tags json1 -o bin/`basename ${PWD}` cli/*.go || make test

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
