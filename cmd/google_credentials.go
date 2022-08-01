package cmd

import "os"

func init() {
	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "service-account-key.json")
}
