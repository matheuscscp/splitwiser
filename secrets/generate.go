package secrets

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

// Generate ...
func Generate() (string, error) {
	const secretSize = 32 // 256 bits
	var buf [secretSize]byte
	n, err := io.ReadFull(rand.Reader, buf[:])
	if err != nil {
		return "", fmt.Errorf("error reading bytes from crypto/rand: %w", err)
	}
	if n != secretSize {
		return "", fmt.Errorf("unexpected number of bytes read from crypto/rand, want %d, got %d", secretSize, n)
	}
	return base64.StdEncoding.EncodeToString(buf[:]), nil
}
