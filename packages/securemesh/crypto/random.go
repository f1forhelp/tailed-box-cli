package crypto

import (
	crand "crypto/rand"
	"errors"
	"fmt"
	"io"
)

var ErrInvalidRandomLength = errors.New("invalid random length")

func RandomBytes(length int) ([]byte, error) {
	return RandomBytesFrom(crand.Reader, length)
}

func RandomBytesFrom(reader io.Reader, length int) ([]byte, error) {
	if reader == nil {
		return nil, errors.New("random reader is required")
	}
	if length <= 0 {
		return nil, fmt.Errorf("%w: %d", ErrInvalidRandomLength, length)
	}

	buf := make([]byte, length)
	if _, err := io.ReadFull(reader, buf); err != nil {
		return nil, err
	}
	return buf, nil
}

func RandomBase32(length int) (string, error) {
	buf, err := RandomBytes(length)
	if err != nil {
		return "", err
	}
	return Base32NoPaddingEncode(buf), nil
}
