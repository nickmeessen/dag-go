package wallet

import "github.com/go-errors/errors"

var (
	// ErrInvalidPrivateKey is returned when a private-key input fails validation:
	// malformed hex, wrong length, or the zero scalar.
	ErrInvalidPrivateKey = errors.New("invalid private key")
)
