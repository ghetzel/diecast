all: vendor fmt build

update:
	-rm -rf vendor
	govend -u

vendor:
	go list github.com/govend/govend
	govend -v -l

fmt:
	gofmt -w .

build:
	go build -o bin/`basename ${PWD}` cli/*.go
