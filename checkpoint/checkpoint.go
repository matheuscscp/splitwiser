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
		Store(v interface{}) error
		Load(v interface{}) error
		Delete() error
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
func NewService(bucket string) (Service, error) {
	client, err := storage.NewClient(context.Background())
	if err != nil {
		return nil, fmt.Errorf("error creating cloud storage client: %w", err)
	}
	return &service{
		client: client.Bucket(bucket).Object("checkpoint.yml"),
		close:  func() { client.Close() },
	}, nil
}

// Close ...
func (s *service) Close() {
	s.close()
}

// Store ...
func (s *service) Store(v interface{}) error {
	w := s.client.NewWriter(context.Background())
	defer w.Close()

	encoder := yaml.NewEncoder(w)
	encoder.SetIndent(2)
	if err := encoder.Encode(v); err != nil {
		return fmt.Errorf("error marshaling checkpoint: %w", err)
	}

	return nil
}

// Load ...
func (s *service) Load(v interface{}) error {
	r, err := s.client.NewReader(context.Background())
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

// Delete ...
func (s *service) Delete() error {
	return s.client.Delete(context.Background())
}
