package main

import (
	"fmt"
	"os"

	"github.com/matheuscscp/splitwiser/bot"
)

func main() {
	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "gcloud.json")
	if err := bot.Run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
