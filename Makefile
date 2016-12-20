all: vendor fmt build

update:
	-rm -rf vendor
	govend -u

vendor:
	go list github.com/govend/govend
	govend --strict

fmt:
	gofmt -w .

test: fmt
	go test -v .

build:
	go build -o bin/`basename ${PWD}` cli/*.go
