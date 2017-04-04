package util

import (
	"fmt"
	"io/ioutil"
	"time"
)

type testingT interface {
	Fatal(args ...interface{})
}

func ValidateWithTimeout(t testingT, validator func() bool, timeout time.Duration) {
	done := make(chan bool, 1)
	go func() {
		for {
			if validator() {
				done <- true
				break
			} else {
				time.Sleep(time.Millisecond)
			}
		}
	}()

	select {
	case <-done:
		return
	case <-time.After(timeout * time.Millisecond):
		t.Fatal("Waited too long")
	}
}

func LoadJson(t testingT, filename string) string {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Fatal(fmt.Sprintf("Could not load file: %s", filename))
	}
	return string(data[:len(data)])
}
