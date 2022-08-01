package events

import (
	"context"
	"errors"
	"fmt"

	"cloud.google.com/go/pubsub"
)

type (
	// Service ...
	Service interface {
		Publish(ctx context.Context, topicID string, data []byte) (id string, err error)
		Close()
	}

	service struct {
		client *pubsub.Client
	}
)

var (
	// ErrServiceNotConfigured ...
	ErrServiceNotConfigured = errors.New("the pubsub client was not configured with a projectID")
)

// NewService ...
func NewService(ctx context.Context, projectID string) (Service, error) {
	if projectID == "" {
		return &service{}, nil
	}
	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("error creating pubsub client: %w", err)
	}
	return &service{client}, nil
}

func (s *service) Close() {
	if s.client != nil {
		s.Close()
	}
}

func (s *service) Publish(ctx context.Context, topicID string, data []byte) (id string, err error) {
	if s.client == nil {
		return "", ErrServiceNotConfigured
	}
	msg := &pubsub.Message{Data: data}
	id, err = s.client.Topic(topicID).Publish(ctx, msg).Get(ctx)
	if err != nil {
		return "", fmt.Errorf("error publishing pubsub message: %w", err)
	}
	return
}
