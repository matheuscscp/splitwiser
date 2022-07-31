package splitwiser

import (
	"fmt"
	"net/http"
	"os"

	"cloud.google.com/go/pubsub"
	"gopkg.in/yaml.v3"
)

type (
	config struct {
		Token     string `yaml:"token"`
		ProjectID string `yaml:"projectID"`
		TopicID   string `yaml:"topicID"`
	}
)

func readConf() *config {
	confFile := os.Getenv("CONF_FILE")
	b, err := os.ReadFile(confFile)
	if err != nil {
		panic(fmt.Errorf("error reading config file '%s': %w", confFile, err))
	}
	var conf config
	if err := yaml.Unmarshal(b, &conf); err != nil {
		panic(fmt.Errorf("error unmarshaling config: %w", err))
	}
	return &conf
}

// StartBot is an HTTP Cloud Function.
func StartBot(w http.ResponseWriter, r *http.Request) {
	conf := readConf()

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
	fmt.Printf("%+v\n", conf)
	fmt.Printf(`form-token="%s"`, r.PostForm.Get("token"))
	if r.PostForm.Get("token") != conf.Token {
		w.WriteHeader(http.StatusUnauthorized)
		writeHTTP(w, "invalid token")
		return
	}

	// create pubsub client
	ctx := r.Context()
	client, err := pubsub.NewClient(ctx, conf.ProjectID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		writeHTTP(w, fmt.Sprintf("error creating pubsub client: %v", err))
		return
	}
	defer client.Close()

	// publish start msg
	msg := &pubsub.Message{Data: []byte("start")}
	if _, err := client.Topic(conf.TopicID).Publish(ctx, msg).Get(ctx); err != nil {
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
