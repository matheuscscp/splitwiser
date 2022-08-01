package splitwiser

import (
	"net/http"

	"github.com/matheuscscp/splitwiser/internal/startbot"
)

// StartBot is an HTTP Cloud Function.
func StartBot(w http.ResponseWriter, r *http.Request) {
	startbot.Run(w, r)
}
