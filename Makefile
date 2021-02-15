PKGS           := $(shell go list ./... 2> /dev/null | grep -v '/vendor')
LOCALS         := $(shell find . -type f -name '*.go' -not -path "./vendor*/*")
ARTIFACT       ?= bin/diecast2

.EXPORT_ALL_VARIABLES:
GO111MODULE  = on
CGO_ENABLED  = 0

all: go.mod deps fmt build

go.mod:
	go mod init github.com/ghetzel/diecast/v2

fmt:
	go mod tidy
	gofmt -w $(LOCALS)
	go generate ./...
	go vet ./...

deps:
	go get ./...

test: fmt deps
	go test -count=1 $(PKGS)

$(ARTIFACT):
	go build --ldflags '-extldflags "-static"' -ldflags '-s' -o $(ARTIFACT) *.go

build: $(ARTIFACT)

.PHONY: fmt deps build $(ARTIFACT)