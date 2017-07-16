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
