package log

import (
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/getconversio/go-utils/util"
	"github.com/puddingfactory/logentrus"
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
