FROM golang:1.8

# This is similar to the golang-onbuild image but with different paths and
# test-dependencies loaded as well.
RUN mkdir -p /go/src/github.com/getconversio/go-utils
WORKDIR /go/src/github.com/getconversio/go-utils

COPY . /go/src/github.com/getconversio/go-utils
RUN go get -v -d -t ./...
