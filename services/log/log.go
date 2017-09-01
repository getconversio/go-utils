package log

import (
	"github.com/getconversio/go-utils/util"
	"github.com/puddingfactory/logentrus"
	log "github.com/sirupsen/logrus"
)

// Setup options for Open Exchange Rates
type Options struct {
	Environment     string
	LogentriesToken string
}

func Setup(options *Options) {
	if options == nil {
		options = &Options{}
	}

	level := log.InfoLevel

	if options.Environment == "development" {
		level = log.DebugLevel
		log.SetFormatter(&log.TextFormatter{ForceColors: true})
	}

	log.SetLevel(level)

	if options.LogentriesToken != "" {
		hook, err := logentrus.New(options.LogentriesToken, &logentrus.Opts{Priority: level})
		util.PanicOnError("Could not create logentries hook", err)
		log.AddHook(hook)
	}
}
