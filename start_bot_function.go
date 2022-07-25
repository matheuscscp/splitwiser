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
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
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
	client, err := pubsub.NewClient(ctx, os.Getenv("PROJECT_ID"))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("error creating pubsub client: %v", err)))
		return
	}
	defer client.Close()

	// publish start msg
	msg := &pubsub.Message{Data: []byte("start")}
	if _, err := client.Topic(os.Getenv("TOPIC_ID")).Publish(ctx, msg).Get(ctx); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("error publishing start message: %v", err)))
		return
	}

	w.Write([]byte("done"))
}
