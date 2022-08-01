package secrets

import (
	"encoding/base64"
	"fmt"
)

func readBinary(secret string) ([]byte, error) {
	b, err := base64.StdEncoding.DecodeString(secret)
	if err != nil {
		return nil, fmt.Errorf("error decoding binary secret from base64: %w", err)
	}
	return b, nil
}
