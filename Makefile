.PHONY: build test
all: test build

build:
	go build -o bin/logstat

test:
	go test