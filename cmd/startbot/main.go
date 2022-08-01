package main

import (
	"net/http"

	_ "github.com/matheuscscp/splitwiser/cmd"
	"github.com/matheuscscp/splitwiser/internal/startbot"
	_ "github.com/matheuscscp/splitwiser/logging"

	"github.com/sirupsen/logrus"
)

func main() {
	err := http.ListenAndServe("localhost:8080", http.HandlerFunc(startbot.Run))
	if err != nil {
		logrus.Fatalf("error on ListenAndServe(): %v", err)
	}
}
