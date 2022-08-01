package main

import (
	"net/http"

	_ "github.com/matheuscscp/splitwiser/cmd"
	_ "github.com/matheuscscp/splitwiser/logging"
	"github.com/matheuscscp/splitwiser/secrets"
	"github.com/matheuscscp/splitwiser/startbot"

	"github.com/sirupsen/logrus"
)

func main() {
	secretsService := secrets.NewMockService()
	err := http.ListenAndServe("localhost:8080", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctxWithSecretsService := secrets.ContextWithService(r.Context(), secretsService)
		reqWithContext := r.WithContext(ctxWithSecretsService)
		startbot.Run(w, reqWithContext)
	}))
	if err != nil {
		logrus.Fatalf("error on ListenAndServe(): %v", err)
	}
}
