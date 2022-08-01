package main

import (
	"net/http"
	"os"

	_ "github.com/matheuscscp/splitwiser/logging"
	"github.com/matheuscscp/splitwiser/secrets"
	"github.com/matheuscscp/splitwiser/startbot"

	"github.com/sirupsen/logrus"
)

func main() {
	jwtSecret, err := secrets.Generate()
	if err != nil {
		logrus.Fatalf("error generating jwt secret: %v", err)
	}
	logrus.Infof("jwt secret: %s", jwtSecret)
	os.Setenv(startbot.JWTSecretEnv, jwtSecret)
	err = http.ListenAndServe("localhost:8080", http.HandlerFunc(startbot.Run))
	if err != nil {
		logrus.Fatalf("error on ListenAndServe(): %v", err)
	}
}
