package rotatesecret

import (
	"context"
	"fmt"
	"hash/crc32"

	_ "github.com/matheuscscp/splitwiser/logging"
	"github.com/matheuscscp/splitwiser/secrets"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"github.com/sirupsen/logrus"
	secretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
)

// Run rotates the given secret.
func Run(ctx context.Context, secretID string) error {
	// create client
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("error creating secretmanager client: %w", err)
	}
	defer client.Close()

	// find previous version
	latest, err := client.GetSecretVersion(ctx, &secretmanagerpb.GetSecretVersionRequest{
		Name: fmt.Sprintf("%s/versions/latest", secretID),
	})
	if err != nil {
		logrus.Warnf("error fetching latest secret version: %v", err)
		latest = nil
	}

	// generate secret and crc32 checksum
	secret, err := secrets.Generate()
	if err != nil {
		return fmt.Errorf("error generating secret: %w", err)
	}
	secretPayload := []byte(secret)
	checksum := int64(crc32.Checksum(secretPayload, crc32.MakeTable(crc32.Castagnoli)))

	// add version
	newVersion, err := client.AddSecretVersion(ctx, &secretmanagerpb.AddSecretVersionRequest{
		Parent: secretID,
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

	logrus.Infof("secret rotated: %s", newVersion.Name)
	return nil
}
