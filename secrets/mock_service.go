package secrets

import (
	"context"
	"sync"
)

type (
	mockService struct {
		cache   map[string]string
		cacheMu sync.Mutex
	}
)

// NewMockService ...
func NewMockService() Service {
	return &mockService{cache: make(map[string]string)}
}

func (m *mockService) Close() {
}

func (m *mockService) Read(ctx context.Context, id string) (string, error) {
	m.cacheMu.Lock()
	defer m.cacheMu.Unlock()

	if c, ok := m.cache[id]; ok {
		return c, nil
	}

	secret, err := Generate()
	if err != nil {
		return "", err
	}
	m.cache[id] = secret

	return secret, nil
}

func (m *mockService) ReadBinary(ctx context.Context, id string) ([]byte, error) {
	secret, _ := m.Read(ctx, id)
	return readBinary(secret)
}
