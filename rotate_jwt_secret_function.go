package splitwiser

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"hash/crc32"
	"io"
	"os"

	_ "github.com/matheuscscp/splitwiser/logging"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"github.com/sirupsen/logrus"
	secretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
)

// RotateJWTSecret is a Pub/Sub Cloud Function.
func RotateJWTSecret(ctx context.Context, m PubSubMessage) error {
	if m.Attributes.EventType != "SECRET_ROTATE" {
		logrus.Infof("event type is not secret rotation: %s", m.Attributes.EventType)
		return nil
	}

	// create client
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("error creating secretmanager client: %w", err)
	}
	defer client.Close()

	// find previous version
	parent := os.Getenv("PARENT")
	latest, err := client.GetSecretVersion(ctx, &secretmanagerpb.GetSecretVersionRequest{
		Name: fmt.Sprintf("%s/versions/latest", parent),
	})
	if err != nil {
		logrus.Warnf("error fetching latest secret version: %v", err)
		latest = nil
	}

	// generate secret and crc32 checksum
	const secretSize = 256
	var buf [secretSize]byte
	n, err := io.ReadFull(rand.Reader, buf[:])
	if err != nil {
		return fmt.Errorf("error reading bytes from crypto/rand: %w", err)
	}
	if n != secretSize {
		return fmt.Errorf("unexpected number of bytes read from crypto/rand, want %d, got %d", secretSize, n)
	}
	secret := base64.StdEncoding.EncodeToString(buf[:])
	secretPayload := []byte(secret)
	checksum := int64(crc32.Checksum(secretPayload, crc32.MakeTable(crc32.Castagnoli)))

	// add version
	_, err = client.AddSecretVersion(ctx, &secretmanagerpb.AddSecretVersionRequest{
		Parent: parent,
		Payload: &secretmanagerpb.SecretPayload{
			Data:       secretPayload,
			DataCrc32C: &checksum,
		},
	})
	if err != nil {
		return fmt.Errorf("error adding secret version: %w", err)
	}

	// destroy previous version
	if latest != nil {
		_, err = client.DestroySecretVersion(ctx, &secretmanagerpb.DestroySecretVersionRequest{
			Name: latest.Name,
		})
		if err != nil {
			return fmt.Errorf("error destroying previous secret version: %w", err)
		}
	}

	return nil
}
