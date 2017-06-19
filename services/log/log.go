package log

import (
	"os"

	"github.com/getconversio/go-utils/util"
	"github.com/getconversio/logentrus"
	log "github.com/sirupsen/logrus"
)

func Setup() {
	level := log.InfoLevel

	if os.Getenv("ENV") == "development" {
		level = log.DebugLevel
		log.SetFormatter(&log.TextFormatter{ForceColors: true})
	}

	log.SetLevel(level)

	if token := os.Getenv("LOGENTRIES_TOKEN"); token != "" {
		hook, err := logentrus.New(token, &logentrus.Opts{Priority: level})
		util.PanicOnError("Could not create logentries hook", err)
		log.AddHook(hook)
	}
}
