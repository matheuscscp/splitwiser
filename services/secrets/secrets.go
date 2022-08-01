package secrets

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"strconv"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"github.com/sirupsen/logrus"
	secretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
)

type (
	// Service ...
	Service interface {
		Read(ctx context.Context, id string) ([]byte, error)
		Rotate(ctx context.Context, id string) error
		Close()
	}

	service struct {
		client *secretmanager.Client
	}
)

var (
	// ErrNilSecretPayload ...
	ErrNilSecretPayload = errors.New("nil secret payload")

	// ErrNilSecretChecksum ...
	ErrNilSecretChecksum = errors.New("nil secret checksum")

	// ErrNilSecretLabels ...
	ErrNilSecretLabels = errors.New("nil secret labels")

	// ErrSecretNumBytesMissing ...
	ErrSecretNumBytesMissing = errors.New("secret label 'num-bytes' is not present")
)

// NewService ...
func NewService(ctx context.Context) (Service, error) {
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("error creating secret manager client: %w", err)
	}
	return &service{client}, nil
}

func (s *service) Close() {
	s.client.Close()
}

func (s *service) Read(ctx context.Context, id string) ([]byte, error) {
	resp, err := s.client.AccessSecretVersion(ctx, &secretmanagerpb.AccessSecretVersionRequest{
		Name: fmt.Sprintf("%s/versions/latest", id),
	})
	if err != nil {
		return nil, fmt.Errorf("error accessing secret version: %w", err)
	}
	payload := resp.GetPayload()
	if payload == nil {
		return nil, ErrNilSecretPayload
	}
	if payload.DataCrc32C == nil {
		return nil, ErrNilSecretChecksum
	}
	want := payload.GetDataCrc32C()
	got := int64(crc32.Checksum(payload.Data, crc32.MakeTable(crc32.Castagnoli)))
	if want != got {
		return nil, fmt.Errorf("secret checksum mismatch, want %v, got %v", want, got)
	}
	secret := payload.GetData()
	b, err := base64.StdEncoding.DecodeString(string(secret))
	if err != nil {
		return nil, fmt.Errorf("error decoding binary secret from base64: %w", err)
	}
	return b, nil
}

func (s *service) Rotate(ctx context.Context, id string) error {
	// fetch number of bytes from secret labels
	secret, err := s.client.GetSecret(ctx, &secretmanagerpb.GetSecretRequest{
		Name: id,
	})
	if err != nil {
		return fmt.Errorf("error getting secret '%s': %w", id, err)
	}
	labels := secret.GetLabels()
	if labels == nil {
		return ErrNilSecretLabels
	}
	secretNumBytes, ok := labels["num-bytes"]
	if !ok {
		return ErrSecretNumBytesMissing
	}
	numBytes, err := strconv.Atoi(secretNumBytes)
	if err != nil {
		return fmt.Errorf("error parsing 'num-bytes' label for secret '%s': %w", id, err)
	}

	// generate random secret
	buf := make([]byte, numBytes)
	n, err := io.ReadFull(rand.Reader, buf)
	if err != nil {
		return fmt.Errorf("error reading %d bytes from crypto/rand: %w", numBytes, err)
	}
	if n != numBytes {
		return fmt.Errorf("unexpected number of bytes read from crypto/rand, want %d, got %d", numBytes, n)
	}
	payload := []byte(base64.StdEncoding.EncodeToString(buf))
	checksum := int64(crc32.Checksum(payload, crc32.MakeTable(crc32.Castagnoli)))

	// find previous version
	latest, err := s.client.GetSecretVersion(ctx, &secretmanagerpb.GetSecretVersionRequest{
		Name: fmt.Sprintf("%s/versions/latest", id),
	})
	if err != nil {
		logrus.Warnf("error fetching latest version of secret '%s': %v", id, err)
		latest = nil
	}

	// add version
	newVersion, err := s.client.AddSecretVersion(ctx, &secretmanagerpb.AddSecretVersionRequest{
		Parent: id,
		Payload: &secretmanagerpb.SecretPayload{
			Data:       payload,
			DataCrc32C: &checksum,
		},
	})
	if err != nil {
		return fmt.Errorf("error adding secret version: %w", err)
	}

	// destroy previous version
	if latest != nil {
		_, err = s.client.DestroySecretVersion(ctx, &secretmanagerpb.DestroySecretVersionRequest{
			Name: latest.Name,
		})
		if err != nil {
			return fmt.Errorf("error destroying previous secret version: %w", err)
		}
	}

	logrus.Infof("secret rotated: %s", newVersion.Name)
	return nil
}
