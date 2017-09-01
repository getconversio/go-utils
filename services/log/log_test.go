package log

import (
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestSetup(t *testing.T) {
	// Should use INFO level by default
	Setup(nil)
	assert.Equal(t, log.InfoLevel, log.GetLevel())
	assert.Empty(t, log.StandardLogger().Hooks)

	// Should use DEBUG level in dev
	options := Options{Environment: "development"}
	Setup(&options)
	assert.Equal(t, log.DebugLevel, log.GetLevel())
	assert.Empty(t, log.StandardLogger().Hooks)

	// Should add logentries hook if token is present
	options = Options{LogentriesToken: "abcd"}
	Setup(&options)
	assert.NotEmpty(t, log.StandardLogger().Hooks)
}
