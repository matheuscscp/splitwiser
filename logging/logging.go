package logging

import (
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

func init() {
	if os.Getenv("DEBUG") != "" {
		logrus.SetLevel(logrus.DebugLevel)
	}
	if l := os.Getenv("LOGRUS_LEVEL"); l != "" {
		level, err := logrus.ParseLevel(l)
		if err != nil {
			logrus.SetLevel(logrus.InfoLevel)
		} else {
			logrus.SetLevel(level)
		}
	}
	logrus.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
	})
}
