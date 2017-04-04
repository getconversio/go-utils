package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockT struct {
	messages []interface{}
}

func (t *mockT) Fatal(args ...interface{}) {
	for _, a := range args {
		t.messages = append(t.messages, a)
	}
}

func TestValidateWithTimeout(t *testing.T) {
	mt := mockT{}
	ValidateWithTimeout(&mt, func() bool { return false }, 1)
	assert.Len(t, mt.messages, 1)
	assert.Contains(t, mt.messages, "Waited too long")

	mt = mockT{}
	ValidateWithTimeout(&mt, func() bool { return true }, 1)
	assert.Len(t, mt.messages, 0)
}

func TestLoadJson(t *testing.T) {
	mt := mockT{}
	LoadJson(&mt, "notthere.json")
	assert.Len(t, mt.messages, 1)
	assert.Contains(t, mt.messages, "Could not load file: notthere.json")
}
