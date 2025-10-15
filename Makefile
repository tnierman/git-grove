DEFAULT := all

.PHONY: all
all: fmt test build

fmt:
	go fmt ./...

test:
	go test ./...

build:
	mkdir -p ./out/
	go build -o ./out/grove .
