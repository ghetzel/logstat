.PHONY: build test
build:
	go build -o bin/logstat

all: gotest build integration print

gotest:
	go test

integration:
	@cat test/test.log       | \
		./bin/logstat -L quiet | \
		tail -n1               | \
		grep 'api 	79 	2xx=60 3xx=14 4xx=3 5xx=2' > /dev/null

	@echo "Running alert threshold test (should take about 30 seconds)..."
	@test -d tmp || mkdir tmp
	@test -f tmp/alert.out && rm tmp/alert.out || exit 0

# generate logs at a known rate that varies on a predictable timescale
# the resulting alerts *should* be predictable based on this input
#
	@./contrib/timed-log-generator.sh 2>>tmp/alert.out | \
		./bin/logstat --requests-max-hits 10 --request-hits-history 5 2>>tmp/alert.out 1>/dev/null


# check the log output for an expected sequence showing the spike in logging rate,
#	followed by an alert
#
	@grep -A3 "^Spike:" tmp/alert.out | \
		grep "High traffic.*hits =" > /dev/null && \
			echo "  High traffic threshold reached at expected location" || \
			(echo "  High traffic did not alert, see './test/alert.out'" && exit 1)

# check the log output for an expected sequence showing the logging rate slowing down,
# followed by the alert clearing
#
	@grep -A3 "^Cool down:" tmp/alert.out | \
		grep "Traffic has returned to normal levels - hits =" > /dev/null && \
		echo "  Traffic threshold cleared at expected location" || \
		(echo "  High traffic did not clear, see './tmp/alert.out'" && exit 1)

print:
	@test -x bin/logstat
	@echo ""
	@echo ""
	@echo "------------------------------------------------------------------------------"
	@echo "'logstat' built successfully and is ready to use at './bin/logstat'"
	@echo ""