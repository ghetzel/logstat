.PHONY: build test
all: test build integration print

build:
	go build -o bin/logstat

test:
	go test

integration:
	@cat test/test.log | ./bin/logstat -L quiet | tail -n1 | grep 'api 	79 	2xx=60 3xx=14 4xx=3 5xx=2' > /dev/null

print:
	@test -x bin/logstat
	@echo ""
	@echo ""
	@echo "------------------------------------------------------------------------------"
	@echo "'logstat' built successfully and is ready to use at './bin/logstat'"
	@echo ""