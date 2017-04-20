#!/usr/bin/env bash

set -e

ci() {
    echo "" > coverage.txt

    for d in $(go list ./...); do
        go test -coverprofile=profile.out $d
        if [ -f profile.out ]; then
            cat profile.out >> coverage.txt
            rm profile.out
        fi
    done
}

test() {
    echo "Running tests"
    go test -cover ./...
    exit $?
}

race() {
    echo "Running tests with race check"
    go test -race ./...
}

cover() {
    echo "Running coverage"
    export PKGS=$(go list ./... | grep -v -e node_modules -e vendor)
    export PKGS_DELIM=$(echo "$PKGS" | paste -sd "," -)
    go list -f '{{if or (len .TestGoFiles) (len .XTestGoFiles)}}go test -covermode count -coverprofile {{.Name}}_{{len .Imports}}_{{len .Deps}}.coverprofile -coverpkg $PKGS_DELIM {{.ImportPath}}{{end}}' $PKGS | xargs -I {} bash -c {}
    gocovmerge $(ls *.coverprofile) > cover.out
    rm *.coverprofile
    go tool cover -html cover.out -o coverage.html
}

"$@"
