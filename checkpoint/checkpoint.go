package checkpoint

import (
	"context"
	"errors"
	"fmt"

	"cloud.google.com/go/storage"
	"gopkg.in/yaml.v3"
)

type (
	// Manager ...
	Manager interface {
		StoreCheckpoint(v interface{}) error
		LoadCheckpoint(v interface{}) error
		DeleteCheckpoint() error
		Close()
	}

	manager struct {
		client *storage.ObjectHandle
		close  func()
	}
)

var (
	// ErrCheckpointNotExist ...
	ErrCheckpointNotExist = errors.New("checkpoint does not exist")
)

// NewManager ...
func NewManager() (Manager, error) {
	client, err := storage.NewClient(context.Background())
	if err != nil {
		return nil, fmt.Errorf("error creating cloud storage client: %w", err)
	}
	return &manager{
		client: client.Bucket("splitwiser-checkpoint").Object("checkpoint.yml"),
		close:  func() { client.Close() },
	}, nil
}

// Close ...
func (m *manager) Close() {
	m.close()
}

// StoreCheckpoint ...
func (m *manager) StoreCheckpoint(v interface{}) error {
	w := m.client.NewWriter(context.Background())
	defer w.Close()

	encoder := yaml.NewEncoder(w)
	encoder.SetIndent(2)
	if err := encoder.Encode(v); err != nil {
		return fmt.Errorf("error marshaling checkpoint: %w", err)
	}

	return nil
}

// LoadCheckpoint ...
func (m *manager) LoadCheckpoint(v interface{}) error {
	r, err := m.client.NewReader(context.Background())
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

// DeleteCheckpoint ...
func (m *manager) DeleteCheckpoint() error {
	return m.client.Delete(context.Background())
}
