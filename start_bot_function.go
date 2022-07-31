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
		writeHTTP(w, `<!DOCTYPE html>
<html>
	<body>
		<form action="/StartBot" method="post">
			<label for="token">Token:</label><br>
			<input type="password" id="token" name="token"><br>
			<input type="submit" value="Submit">
		</form>
	</body>
</html>
`)
		return
	}

	// read token and validate
	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		writeHTTP(w, fmt.Sprintf("error parsing form: %v", err))
		return
	}
	if r.PostForm.Get("token") != os.Getenv("TOKEN") {
		w.WriteHeader(http.StatusUnauthorized)
		writeHTTP(w, "invalid token")
		return
	}

	// create pubsub client
	ctx := r.Context()
	client, err := pubsub.NewClient(ctx, os.Getenv("PROJECT_ID"))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		writeHTTP(w, fmt.Sprintf("error creating pubsub client: %v", err))
		return
	}
	defer client.Close()

	// publish start msg
	msg := &pubsub.Message{Data: []byte("start")}
	if _, err := client.Topic(os.Getenv("TOPIC_ID")).Publish(ctx, msg).Get(ctx); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		writeHTTP(w, fmt.Sprintf("error publishing start message: %v", err))
		return
	}

	writeHTTP(w, "done")
}

func writeHTTP(w http.ResponseWriter, resp string) {
	if _, err := w.Write([]byte(resp)); err != nil {
		panic(err)
	}
}
