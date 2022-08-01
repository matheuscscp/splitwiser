package secrets

import (
	"context"
	"fmt"
	"hash/crc32"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	secretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
)

type (
	// Service ...
	Service interface {
		Read(ctx context.Context, id string) (string, error)
		ReadBinary(ctx context.Context, id string) ([]byte, error)
		Close()
	}

	service struct {
		client *secretmanager.Client
	}
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

func (s *service) Read(ctx context.Context, id string) (string, error) {
	resp, err := s.client.AccessSecretVersion(ctx, &secretmanagerpb.AccessSecretVersionRequest{
		Name: fmt.Sprintf("%s/versions/latest", id),
	})
	if err != nil {
		return "", fmt.Errorf("error accessing secret version: %w", err)
	}
	payload := resp.GetPayload()
	if payload.DataCrc32C != nil {
		want := payload.GetDataCrc32C()
		got := int64(crc32.Checksum(payload.Data, crc32.MakeTable(crc32.Castagnoli)))
		if want != got {
			return "", fmt.Errorf("secret checksum mismatch, want %v, got %v", want, got)
		}
	}
	return string(payload.GetData()), nil
}

func (s *service) ReadBinary(ctx context.Context, id string) ([]byte, error) {
	secret, err := s.Read(ctx, id)
	if err != nil {
		return nil, err
	}
	return readBinary(secret)
}
