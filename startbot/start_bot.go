package startbot

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	_ "github.com/matheuscscp/splitwiser/logging"
	"github.com/matheuscscp/splitwiser/secrets"

	"cloud.google.com/go/pubsub"
	jwt "github.com/golang-jwt/jwt/v4"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

type (
	config struct {
		Password    string `yaml:"password"`
		ProjectID   string `yaml:"projectID"`
		TopicID     string `yaml:"topicID"`
		JWTSecretID string `yaml:"jwtSecretID"`
	}

	controller struct {
		conf      *config
		jwtSecret []byte
		w         http.ResponseWriter
		r         *http.Request
	}
)

const (
	httpHeaderAuthorization = "Authorization"
	httpHeaderContentType   = "Content-Type"
)

var (
	errWrongPassword = errors.New("wrong password")
	errInvalidRealm  = errors.New("invalid authentication realm")
	errInvalidToken  = errors.New("invalid token")

	jwtSigningMethod = jwt.SigningMethodHS256
)

// Run serves the start-bot website.
func Run(w http.ResponseWriter, r *http.Request) {
	conf := readConfig()

	// create secrets client
	ctx := r.Context()
	secretsService, err := secrets.NewService(ctx)
	if err != nil {
		logrus.Fatalf("error creating secrets service: %v", err)
	}
	defer secretsService.Close()

	// read jwt secret
	jwtSecret, err := secretsService.ReadBinary(ctx, conf.JWTSecretID)
	if err != nil {
		logrus.Fatalf("error reading jwt secret: %v", err)
	}
	h := sha256.New()
	h.Write(jwtSecret)
	logrus.Infof("loaded jwt secret. hash: %x", h.Sum(nil))

	(&controller{
		conf:      conf,
		jwtSecret: jwtSecret,
		w:         w,
		r:         r,
	}).handleRequest()
}

func readConfig() *config {
	confFile := os.Getenv("CONF_FILE")
	if confFile == "" {
		confFile = "config.yml"
	}
	b, err := os.ReadFile(confFile)
	if err != nil {
		logrus.Fatalf("error reading config file '%s': %v", confFile, err)
	}
	var conf config
	if err := yaml.Unmarshal(b, &conf); err != nil {
		logrus.Fatalf("error unmarshaling config: %v", err)
	}
	return &conf
}

func (c *controller) handleRequest() {
	// handle get (public)
	if c.r.Method == http.MethodGet {
		c.sendSinglePageApp()
		return
	}
	// handle non-post (not supported)
	if c.r.Method != http.MethodPost {
		c.replyStatusCode(http.StatusMethodNotAllowed)
		return
	}
	// post

	if !c.hasAuthentication() {
		if err := c.checkPassword(); err != nil {
			if errors.Is(err, errWrongPassword) {
				c.replyStatusCode(http.StatusUnauthorized)
			} else {
				c.replyError(http.StatusBadRequest, err)
			}
			return
		}
	} else if err := c.checkAuthentication(); err != nil {
		logrus.Warnf("invalid authentication: %v", err)
		c.replyStatusCode(http.StatusUnauthorized)
		return
	}

	if err := c.startBot(); err != nil {
		c.replyError(http.StatusInternalServerError, err)
		return
	}

	if !c.hasAuthentication() {
		c.sendNewJWT()
	} else {
		c.replyStatusCode(http.StatusCreated)
	}
}

func (c *controller) hasAuthentication() bool {
	return c.r.Header.Get(httpHeaderAuthorization) != ""
}

func (c *controller) sendSinglePageApp() {
	c.w.Header().Set(httpHeaderContentType, "text/html; charset=utf-8")
	c.writeHTTP(`<!DOCTYPE html>
<html>
	<head>
		<script>
			async function startApp() {
				const token = localStorage.getItem('auth_token')
				if (!token) {
					console.log('no token found locally')
					selectDiv('form')
					return
				}
				console.log('token found locally, sending start command...')
				const resp = await fetch(window.location.href, {
					method: 'POST',
					headers: {
						'Authorization': 'Bearer ' + token,
					},
				})
				if (resp.status === 401) {
					console.log('token expired, destroying local copy...')
					localStorage.removeItem('auth_token')
					selectDiv('form')
				} else if (resp.status === 201) {
					success()
				} else {
					showServerError(resp)
				}
			}

			async function submit() {
				const password = document.getElementById('password').value
				console.log('sending start command...')
				selectDiv('loading')
				const resp = await fetch(window.location.href, {
					method: 'POST',
					headers: {
						'Content-Type': 'application/json',
					},
					body: JSON.stringify({ password }),
				})
				if (resp.status === 401) {
					console.log('invalid password')
					showDiv('invalid-password')
					selectDiv('form')
				} else if (resp.status === 201) {
					const { auth_token } = await resp.json()
					localStorage.setItem('auth_token', auth_token)
					success()
				} else {
					showServerError(resp)
				}
			}

			function success() {
				console.log('success')
				hideDiv('server-error')
				selectDiv('success')
			}

			function showServerError(resp) {
				const err = JSON.stringify(resp, null, 2)
				console.log('unexpected server error:', err)
				document.getElementById('server-error').text = err
				showDiv('server-error')
			}

			function showDiv(divID) {
				document.getElementById(divID).removeAttribute('hidden')
			}

			function hideDiv(divID) {
				document.getElementById(divID).setAttribute('hidden', true)
			}

			function selectDiv(divID) {
				['loading', 'form', 'success'].filter(id => id !== divID).forEach(id => {
					hideDiv(id)
				})
				showDiv(divID)
			}
		</script>
	</head>
	<body onload="startApp()">
		<div id="server-error" hidden></div>
		<div id="loading">
			Loading...
		</div>
		<div id="form" hidden>
			<div id="invalid-password" hidden>
				Invalid password.
			</div>
			<label for="password">Password:</label><br>
			<input type="password" id="password" name="password"><br>
			<button onclick="submit()">Submit</button>
		</div>
		<div id="success" hidden>
			Success!
		</div>
	</body>
</html>
`)
}

func (c *controller) sendNewJWT() {
	token := jwt.NewWithClaims(jwtSigningMethod, jwt.MapClaims{
		"exp": time.Now().Add(30 * 24 * time.Hour).Unix(),
	})
	tokenString, err := token.SignedString(c.jwtSecret)
	if err != nil {
		logrus.Errorf("error signing jwt token: %v", err)
		tokenString = "null"
	} else {
		tokenString = fmt.Sprintf(`"%s"`, tokenString)
	}
	c.w.Header().Set(httpHeaderContentType, "application/json")
	c.w.WriteHeader(http.StatusCreated)
	c.writeHTTP(`{"auth_token":%s}`, tokenString)
}

func (c *controller) checkAuthentication() error {
	// fetch token from request
	const realm = "Bearer "
	authn := c.r.Header.Get(httpHeaderAuthorization)
	if !strings.HasPrefix(authn, realm) {
		return errInvalidRealm
	}
	tokenString := authn[len(realm):]

	// validate token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if method, ok := token.Method.(*jwt.SigningMethodHMAC); !ok || method != jwtSigningMethod {
			return nil, fmt.Errorf("invalid signing method: %v", token.Header["alg"])
		}
		return c.jwtSecret, nil
	})
	if err != nil {
		return fmt.Errorf("error parsing jwt token: %w", err)
	}
	if !token.Valid {
		return errInvalidToken
	}
	return nil
}

func (c *controller) checkPassword() error {
	var payload struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(c.r.Body).Decode(&payload); err != nil {
		return fmt.Errorf("error unmarshaling payload: %w", err)
	}
	if payload.Password != c.conf.Password {
		return errWrongPassword
	}
	return nil
}

func (c *controller) startBot() error {
	if c.conf.ProjectID == "" || c.conf.TopicID == "" {
		logrus.Error("cannot publish start-bot event, no project/topic configured")
		return nil
	}
	ctx := c.r.Context()
	client, err := pubsub.NewClient(ctx, c.conf.ProjectID)
	if err != nil {
		return fmt.Errorf("error creating pubsub client: %v", err)
	}
	defer client.Close()
	msg := &pubsub.Message{Data: []byte("start")}
	if _, err := client.Topic(c.conf.TopicID).Publish(ctx, msg).Get(ctx); err != nil {
		return fmt.Errorf("error publishing start message: %v", err)
	}
	return nil
}

func (c *controller) writeHTTP(format string, args ...interface{}) {
	resp := fmt.Sprintf(format, args...)
	if _, err := c.w.Write([]byte(resp)); err != nil {
		logrus.Fatalf("error writing response: %v", err)
	}
}

func (c *controller) replyStatusCode(code int) {
	c.w.WriteHeader(code)
	c.writeHTTP(http.StatusText(code))
}

func (c *controller) replyError(code int, err error) {
	c.w.WriteHeader(code)
	c.writeHTTP(err.Error())
}
