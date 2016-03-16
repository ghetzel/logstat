package util

import (
    "os"
    log "github.com/Sirupsen/logrus"
)

func ParseLogLevel(level string) {
    log.SetOutput(os.Stderr)

    switch level {
    case `info`:
        log.SetLevel(log.InfoLevel)
    case `warn`:
        log.SetLevel(log.WarnLevel)
    case `error`:
        log.SetLevel(log.ErrorLevel)
    case `fatal`:
        log.SetLevel(log.FatalLevel)
    case `quiet`:
        log.SetLevel(log.PanicLevel)
    default:
        log.SetLevel(log.DebugLevel)
    }
}