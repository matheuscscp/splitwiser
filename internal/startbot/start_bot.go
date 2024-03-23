package startbot

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/matheuscscp/splitwiser/config"
	_ "github.com/matheuscscp/splitwiser/logging"
	"github.com/matheuscscp/splitwiser/models"
	"github.com/matheuscscp/splitwiser/services/events"
	"github.com/matheuscscp/splitwiser/services/secrets"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"
)

type (
	controller struct {
		conf          *config.StartBot
		w             http.ResponseWriter
		r             *http.Request
		eventsService events.Service
	}
)

const (
	httpHeaderAuthorization = "Authorization"
	httpHeaderContentType   = "Content-Type"
)

var (
	errInvalidUser     = errors.New("invalid user")
	errInvalidPassword = errors.New("invalid password")
	errInvalidRealm    = errors.New("invalid authentication realm")
	errInvalidToken    = errors.New("invalid token")

	jwtSigningMethod = jwt.SigningMethodHS256
)

// Run serves the start-bot website.
func Run(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// load config
	var conf config.StartBot
	if err := config.Load(&conf); err != nil {
		logrus.Fatalf("error loading config: %v", err)
	}

	// create secrets service
	secretsService, err := secrets.NewService(ctx)
	if err != nil {
		logrus.Fatalf("error creating secrets service: %v", err)
	}
	defer secretsService.Close()

	// read jwt secret
	conf.JWTSecret, err = secretsService.Read(ctx, conf.JWTSecretID)
	if err != nil {
		logrus.Fatalf("error reading jwt secret: %v", err)
	}

	// create events service
	eventsService, err := events.NewService(ctx, conf.ProjectID)
	if err != nil {
		logrus.Fatalf("error creating events service: %v", err)
	}
	defer eventsService.Close()

	(&controller{
		conf:          &conf,
		w:             w,
		r:             r,
		eventsService: eventsService,
	}).handleRequest()
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

	var user models.ReceiptItemOwner
	var err error
	if !c.hasAuthentication() {
		if user, err = c.checkUserAndPassword(); err != nil {
			switch {
			case errors.Is(err, errInvalidUser):
				logrus.Warn("invalid user")
				c.replyStatusCode(http.StatusUnprocessableEntity)
			case errors.Is(err, errInvalidPassword):
				logrus.Warn("invalid password")
				c.replyStatusCode(http.StatusUnauthorized)
			default:
				c.replyError(http.StatusBadRequest, err)
			}
			return
		}
	} else if user, err = c.checkAuthentication(); err != nil {
		logrus.Warnf("invalid authentication: %v", err)
		c.replyStatusCode(http.StatusUnauthorized)
		return
	}

	if err := c.startBot(user); err != nil {
		c.replyError(http.StatusInternalServerError, err)
		return
	}

	if !c.hasAuthentication() {
		c.sendNewJWT(user)
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
					showError(await resp.text())
				}
			}

			async function submit() {
				const user = document.getElementById('user').value
				const password = document.getElementById('password').value
				console.log('sending start command...')
				selectDiv('loading')
				const resp = await fetch(window.location.href, {
					method: 'POST',
					headers: {
						'Content-Type': 'application/json',
					},
					body: JSON.stringify({ user, password }),
				})
				const { status } = resp
				if (status >= 400 && status < 500) {
					const err = status === 422 ? 'Invalid user.' : 'Invalid password.'
					showError(err)
				} else if (status === 201) {
					const { auth_token } = await resp.json()
					localStorage.setItem('auth_token', auth_token)
					success()
				} else {
					showError(await resp.text())
				}
			}

			function success() {
				console.log('success')
				hideDiv('error-message')
				selectDiv('success')
			}

			function showError(err) {
				console.log(err)
				document.getElementById('error-message').text = err
				showDiv('error-message')
				selectDiv('form')
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
		<div id="loading">
			Loading...
		</div>
		<div id="form" hidden>
			<div id="error-message" hidden></div>

			<label for="user">User ('a' or 'm'):</label><br>
			<input type="text" id="user" name="user"><br>

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

func (c *controller) sendNewJWT(user models.ReceiptItemOwner) {
	token := jwt.NewWithClaims(jwtSigningMethod, jwt.MapClaims{
		"sub": string(user),
		"exp": time.Now().Add(30 * 24 * time.Hour).Unix(),
	})
	tokenString, err := token.SignedString(c.conf.JWTSecret)
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

func (c *controller) checkAuthentication() (models.ReceiptItemOwner, error) {
	// fetch token from request
	const realm = "Bearer "
	authn := c.r.Header.Get(httpHeaderAuthorization)
	if !strings.HasPrefix(authn, realm) {
		return "", errInvalidRealm
	}
	tokenString := authn[len(realm):]

	// validate token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return c.conf.JWTSecret, nil
	}, jwt.WithValidMethods([]string{jwtSigningMethod.Name}))
	if err != nil {
		return "", fmt.Errorf("error parsing jwt token: %w", err)
	}
	if !token.Valid {
		return "", errInvalidToken
	}
	sub, err := token.Claims.GetSubject()
	if err != nil {
		return "", fmt.Errorf("error getting subject from token: %w", err)
	}
	user := models.ReceiptItemOwner(sub)
	if user != models.Ana && user != models.Matheus {
		return "", errInvalidUser
	}
	return user, nil
}

func (c *controller) checkUserAndPassword() (models.ReceiptItemOwner, error) {
	var payload struct {
		User     string `json:"user"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(c.r.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("error unmarshaling payload: %w", err)
	}
	user := models.ReceiptItemOwner(strings.TrimSpace(strings.ToLower(payload.User)))
	if user != models.Ana && user != models.Matheus {
		return "", errInvalidUser
	}
	if payload.Password != c.conf.Password {
		return "", errInvalidPassword
	}
	return user, nil
}

func (c *controller) startBot(user models.ReceiptItemOwner) error {
	msg := fmt.Sprintf("start-%s", user)
	serverID, err := c.eventsService.Publish(c.r.Context(), c.conf.TopicID, []byte(msg))
	if err != nil {
		if errors.Is(err, events.ErrServiceNotConfigured) {
			logrus.Error("cannot publish start-bot event, events service is not configured")
			return nil
		}
		return fmt.Errorf("error publishing start-bot event: %w", err)
	}
	logrus.Infof("start-bot event published with serverID=%s", serverID)
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
	logrus.WithError(err).Errorf("HTTP status code %d", code)
	c.w.WriteHeader(code)
	c.writeHTTP(err.Error())
}
