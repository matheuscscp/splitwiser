package checkpoint

import (
	"context"
	"errors"
	"fmt"

	"cloud.google.com/go/storage"
	"gopkg.in/yaml.v3"
)

type (
	// Service ...
	Service interface {
		Store(ctx context.Context, v interface{}) error
		Load(ctx context.Context, v interface{}) error
		Delete(ctx context.Context) error
		Close()
	}

	service struct {
		client *storage.ObjectHandle
		close  func()
	}
)

var (
	// ErrCheckpointNotExist ...
	ErrCheckpointNotExist = errors.New("checkpoint does not exist")
)

// NewService ...
func NewService(ctx context.Context, bucket string) (Service, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("error creating cloud storage client: %w", err)
	}
	return &service{
		client: client.Bucket(bucket).Object("checkpoint.yml"),
		close:  func() { client.Close() },
	}, nil
}

func (s *service) Close() {
	s.close()
}

func (s *service) Store(ctx context.Context, v interface{}) error {
	w := s.client.NewWriter(ctx)
	defer w.Close()

	encoder := yaml.NewEncoder(w)
	encoder.SetIndent(2)
	if err := encoder.Encode(v); err != nil {
		return fmt.Errorf("error marshaling checkpoint: %w", err)
	}

	return nil
}

func (s *service) Load(ctx context.Context, v interface{}) error {
	r, err := s.client.NewReader(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return ErrCheckpointNotExist
		}
		return fmt.Errorf("error creating checkpoint reader: %w", err)
	}
	defer r.Close()

	if err := yaml.NewDecoder(r).Decode(v); err != nil {
		return fmt.Errorf("error unmarshaling checkpoint: %w", err)
	}

	return nil
}

func (s *service) Delete(ctx context.Context) error {
	return s.client.Delete(ctx)
}
