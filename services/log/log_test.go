package log

import (
	"os"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestSetup(t *testing.T) {
	defer os.Setenv("ENV", os.Getenv("ENV"))
	defer os.Setenv("LOGENTRIES_TOKEN", os.Getenv("LOGENTRIES_TOKEN"))

	// Should use INFO level by default
	Setup()
	assert.Equal(t, log.InfoLevel, log.GetLevel())
	assert.Empty(t, log.StandardLogger().Hooks)

	// Should use DEBUG level in dev
	os.Setenv("ENV", "development")
	Setup()
	assert.Equal(t, log.DebugLevel, log.GetLevel())
	assert.Empty(t, log.StandardLogger().Hooks)

	// Should add logentries hook if token is present
	os.Setenv("LOGENTRIES_TOKEN", "abcd")
	Setup()
	assert.NotEmpty(t, log.StandardLogger().Hooks)
}
