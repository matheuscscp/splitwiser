package splitwiser

import (
	"fmt"
	"net/http"
	"os"

	"cloud.google.com/go/pubsub"
)

// StartBot is an HTTP Cloud Function.
func StartBot(w http.ResponseWriter, r *http.Request) {
	// handle get
	if r.Method == http.MethodGet {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<!DOCTYPE html>
<html>
	<body>
	<form action="/StartBot" method="post">
		<label for="token">Token:</label><br>
		<input type="password" id="token" name="token"><br>
		<input type="submit" value="Submit">
	</form>
	</body>
</html>
`))
		return
	}

	// read token and validate
	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("error parsing form: %v", err)))
		return
	}
	if r.PostForm.Get("token") != os.Getenv("TOKEN") {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("invalid token"))
		return
	}

	// create pubsub client
	ctx := r.Context()
	client, err := pubsub.NewClient(ctx, "splitwiser-356519" /*projectID*/)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("error creating pubsub client: %v", err)))
		return
	}
	defer client.Close()

	// publish start msg
	topic := client.Topic("start-bot" /*id*/)
	pubResult := topic.Publish(ctx, &pubsub.Message{
		Data: []byte("start"),
	})
	messageID, err := pubResult.Get(ctx)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("error publishing start msg: %v", err)))
		return
	}

	w.Write([]byte(fmt.Sprintf("Bot will be started by message ID '%s'.", messageID)))
}
