package splitwiser

import (
	"context"

	"github.com/matheuscscp/splitwiser/internal/rotatesecret"
	_ "github.com/matheuscscp/splitwiser/logging"

	"github.com/sirupsen/logrus"
)

// RotateSecret is a Pub/Sub Cloud Function.
func RotateSecret(ctx context.Context, m PubSubMessage) error {
	if m.Attributes.EventType != "SECRET_ROTATE" {
		logrus.Infof("event %s skipped on secret %s", m.Attributes.EventType, m.Attributes.SecretId)
		return nil
	}

	return rotatesecret.Run(ctx, m.Attributes.SecretId)
}
