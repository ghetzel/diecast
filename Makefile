.PHONY: test deps

PKGS=`go list ./... | grep -v /vendor/`
LOCALS=`find . -type f -name '*.go' -not -path "./vendor/*"`

all: deps fmt test build

deps:
	@go list github.com/mjibson/esc           || go get github.com/mjibson/esc/...
	@go list golang.org/x/tools/cmd/goimports || go get golang.org/x/tools/cmd/goimports
	go generate -x
	go get ./...

clean:
	-rm -rf bin

fmt:
	goimports -w $(LOCALS)
	go vet $(PKGS)

test:
	go test $(PKGS)

build: deps fmt
	test -d cli && go build -o bin/`basename ${PWD}` cli/main.go
	test -d cli/funcdoc && go build -o bin/funcdoc cli/funcdoc/main.go

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
