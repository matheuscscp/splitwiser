package splitwiser

type (
	// PubSubMessage is the payload of a Pub/Sub event.
	PubSubMessage struct {
		Attributes PubSubAttributes `json:"attributes"`
		Data       []byte           `json:"data"`
	}

	// PubSubAttributes are attributes from the Pub/Sub event.
	PubSubAttributes struct {
		EventType string `json:"eventType"`
	}
)
