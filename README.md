# go-utils

A collection of utilities function for Go used across Conversio. No reason this
can't be open source right? :-)

[![Build Status](https://travis-ci.org/getconversio/go-utils.svg?branch=master)](https://travis-ci.org/getconversio/go-utils)
[![codecov](https://codecov.io/gh/getconversio/go-utils/branch/master/graph/badge.svg)](https://codecov.io/gh/getconversio/go-utils)

## Usage

```go
import "github.com/getconversio/go-utils/services/amqp"

func StartListening() {
	amqp.HandleFunc(
		"myqueue",     // Queue name
		"myexchange",  // Exchange name
		"myrouting",   // Routing key
		new(mystruct),
		func(msg interface{}, headers aq.Table) error {
			doSomething(*msg.(*mystruct))
			return nil
		})
}
```
