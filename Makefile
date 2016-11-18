all: vendor fmt build

update:
	glide up

vendor:
	go list github.com/Masterminds/glide
	glide install

fmt:
	gofmt -w .

build:
	go build -o bin/`basename ${PWD}`
