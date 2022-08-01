package main

import (
	"context"
	"fmt"
	"os"

	_ "github.com/matheuscscp/splitwiser/cmd"
	"github.com/matheuscscp/splitwiser/internal/rotatesecret"
	_ "github.com/matheuscscp/splitwiser/logging"

	"github.com/sirupsen/logrus"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("Usage: %s SECRET_ID\n", os.Args[0])
		return
	}

	secretID := os.Args[1]
	if err := rotatesecret.Run(context.Background(), secretID); err != nil {
		logrus.Fatalf("error rotating secret '%s': %v", secretID, err)
	}
}
