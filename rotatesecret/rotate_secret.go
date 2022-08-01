package rotatesecret

import (
	"context"
	"fmt"

	"github.com/matheuscscp/splitwiser/services/secrets"
)

// Run rotates the given secret.
func Run(ctx context.Context, secretID string) error {
	secretsService, err := secrets.NewService(ctx)
	if err != nil {
		return fmt.Errorf("error creating secrets service: %w", err)
	}
	defer secretsService.Close()
	return secretsService.Rotate(ctx, secretID)
}
