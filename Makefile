.PHONY: deps fmt build

all: fmt deps build

deps:
	go get .

fmt:
	gofmt -w .

test: fmt
	go test .

build:
	go build -o bin/`basename ${PWD}` cli/*.go
