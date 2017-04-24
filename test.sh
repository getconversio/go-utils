#!/usr/bin/env bash

set -e

ci() {
    echo "" > coverage.txt

    for d in $(go list ./... | grep -v -e cmd); do
        go test -coverprofile=profile.out $d
        if [ -f profile.out ]; then
            cat profile.out >> coverage.txt
            rm profile.out
        fi
    done
}

test() {
    echo "Running tests"
    go test -cover $(go list ./... | grep -v -e cmd)
    exit $?
}

race() {
    echo "Running tests with race check"
    go test -race $(go list ./... | grep -v -e cmd)
}

cover() {
    echo "Running coverage"
    rm coverage.txt
    rm coverage.html
    echo 'mode: atomic' > coverage.txt
    go list ./... | grep -v -e cmd | xargs -n1 -I{} sh -c 'go test -covermode=count -coverprofile=coverage.tmp {} && tail -n +2 coverage.tmp >> coverage.txt'
    rm coverage.tmp
    go tool cover -html coverage.txt -o coverage.html
}

"$@"
