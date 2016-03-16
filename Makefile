all: build

build:
	go build -o bin/logstat

test:
	go test