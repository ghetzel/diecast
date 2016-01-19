all: build

build:
	./build.sh
debug:
	DEBUG=true ./build.sh

clean:
	git clean -fd