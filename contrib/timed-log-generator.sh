#!/usr/bin/env bash

# generate 35 lines at 5 logs/second
echo "Warm-up: 35 @ 5/sec" 1>&2
./$(dirname "${BASH_SOURCE}")/log-generator.sh 35 0.2

# generate 200 lines at 20 logs/second
echo "Spike: 200 @ 20/sec" 1>&2
./$(dirname "${BASH_SOURCE}")/log-generator.sh 200 0.05

# generate 50 lines at 5 logs/second
echo "Cool down: 50 @ 5/sec" 1>&2
./$(dirname "${BASH_SOURCE}")/log-generator.sh 50 0.2