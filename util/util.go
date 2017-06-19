package util

import (
	"fmt"
	"hash/fnv"
	"io"
	"io/ioutil"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"syscall"

	log "github.com/sirupsen/logrus"
)

func Hash32(s string) string {
	h := fnv.New32()
	io.WriteString(h, s)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func PanicOnError(msg string, err error) {
	if err != nil {
		log.Panic(msg, err)
	}
}

func GetenvInt(key string, fallback int) int {
	s := os.Getenv(key)
	if v, err := strconv.Atoi(s); err == nil {
		return v
	}
	return fallback
}

func Getenv(key, fallback string) string {
	if s := os.Getenv(key); s != "" {
		return s
	}
	return fallback
}

func GracefulShutdown(cleanupFun func()) chan bool {
	log.Debug("Setting up OS signals")
	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)

	signal.Notify(sigs, os.Interrupt, os.Kill, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		sig := <-sigs
		log.Infof("Gracefully shutting down after signal: %s", sig)
		cleanupFun()
		done <- true
	}()

	return done
}

var envPattern = regexp.MustCompile(`\$[A-Z_]+`)

// Read a file (e.g. a configuration file) and replace any environment variable
// placeholders with their environment counterparts. This is similar to the bash command envsubst
//
// For example, if the file contains: {"key":"$VAR"} then $VAR will be replaced
// with the environment variable VAR.  Only supports environment variables with
// uppercaser characters and underscore.
func ReadFileEnvsubst(path string) string {
	data, err := ioutil.ReadFile(path)
	PanicOnError("Could not read config file", err)
	result := envPattern.ReplaceAllStringFunc(string(data), func(match string) string {
		// A match here is e.g. $MYVAR
		// Go does not support positive lookbehind and lookahead so the $ will
		// have to be removed manually here before looking up the environment
		// variable.
		match = strings.Replace(match, "$", "", -1)
		return fmt.Sprintf("%s", os.Getenv(match))
	})
	return result
}
