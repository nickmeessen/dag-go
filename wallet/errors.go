package wallet

import "github.com/go-errors/errors"

var (
	// ErrInvalidPrivateKey is returned when a private-key input fails validation:
	// malformed hex, wrong length, or the zero scalar.
	ErrInvalidPrivateKey = errors.New("invalid private key")
	// ErrInvalidMnemonic is returned when a supplied BIP39 phrase is malformed —
	// wrong word count, unknown words, or a checksum mismatch.
	ErrInvalidMnemonic = errors.New("invalid mnemonic")
)
