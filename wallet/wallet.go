// Package wallet provides key generation, address derivation, and
// transaction signing for the Constellation (DAG) network.
//
// A Wallet wraps a secp256k1 private key and exposes the two public
// identities Constellation uses: the DAG address (on-chain, 40 chars) and
// the peer ID (node identifier, 128 hex chars).
package wallet

import (
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"
	"github.com/mr-tron/base58"
	"github.com/nickmeessen/dag-go/tx"
)

// pkcsPrefix is the 23-byte DER-encoded SubjectPublicKeyInfo header for
// secp256k1. Prepending it to the raw uncompressed public key produces the
// PKIX serialization that Constellation's address algorithm hashes.
var pkcsPrefix = mustHex("3056301006072a8648ce3d020106052b8104000a034200")

// Wallet holds a secp256k1 private key and derives Constellation-specific
// public identities (address, peer ID) from it on demand.
type Wallet struct {
	privateKey *secp256k1.PrivateKey
}

// New generates a wallet backed by a fresh cryptographically secure secp256k1 private key.
func New() (*Wallet, error) {
	priv, err := secp256k1.GeneratePrivateKey()
	if err != nil {
		return nil, fmt.Errorf("generate private key: %w", err)
	}
	return &Wallet{privateKey: priv}, nil
}

// FromPrivateKey constructs a wallet from a 32-byte hex-encoded private key.
func FromPrivateKey(privateKey string) (*Wallet, error) {
	keyBytes, err := hex.DecodeString(privateKey)
	if err != nil {
		return nil, fmt.Errorf("%w: hex decode: %w", ErrInvalidPrivateKey, err)
	}
	if len(keyBytes) != 32 {
		return nil, fmt.Errorf("%w: expected 32 bytes, got %d", ErrInvalidPrivateKey, len(keyBytes))
	}

	priv := secp256k1.PrivKeyFromBytes(keyBytes)

	// PrivKeyFromBytes silently reduces values >= N mod N and accepts zero.
	// Reject zero explicitly — it's not a valid secp256k1 scalar.
	if priv.Key.IsZero() {
		return nil, fmt.Errorf("%w: zero scalar", ErrInvalidPrivateKey)
	}

	return &Wallet{privateKey: priv}, nil
}

// PrivateKeyHex returns the 32-byte private key as 64 hex characters.
// Treat the returned value as secret, leaking it lets anyone sign for this wallet.
func (w *Wallet) PrivateKeyHex() string {
	return hex.EncodeToString(w.privateKey.Serialize())
}

// Address returns the DAG address derived from the wallet's public key.
// The result is always 40 characters: "DAG" + one parity digit + 36 base58 characters.
func (w *Wallet) Address() string {
	pubBytes := w.publicKeyBytes()

	preimage := make([]byte, 0, len(pkcsPrefix)+len(pubBytes))
	preimage = append(preimage, pkcsPrefix...)
	preimage = append(preimage, pubBytes...)

	sum := sha256.Sum256(preimage)
	encoded := base58.Encode(sum[:])
	last36 := encoded[len(encoded)-36:]

	parity := 0
	for i := 0; i < len(last36); i++ {
		if c := last36[i]; c >= '0' && c <= '9' {
			parity += int(c - '0')
		}
	}
	parity %= 9

	return fmt.Sprintf("DAG%d%s", parity, last36)
}

// PeerID returns the uncompressed public key without the 04 prefix as 128 hex characters.
// Used as the identifier in transaction proofs and validator registration.
func (w *Wallet) PeerID() string {
	return hex.EncodeToString(w.publicKeyBytes()[1:])
}

func (w *Wallet) publicKeyBytes() []byte {
	return w.privateKey.PubKey().SerializeUncompressed()
}

// Sign produces a signed transaction from the given Transfer. A zero Salt
// triggers fresh salt generation — callers wanting deterministic output
// (e.g. for tests) should set Salt explicitly.
//
// The signing chain is: canonical Encode → KryoSerialize → SHA-256 → SHA-512
// over the hex-encoded SHA-256 output → ECDSA with RFC6979 → DER-encoded,
// hex-encoded into the proof. The double-hash with an intermediate hex
// conversion is protocol-mandated; don't "simplify" it to a single hash.
func (w *Wallet) Sign(t tx.Transfer) (tx.Signed, error) {
	if t.Salt == 0 {
		salt, err := tx.GenerateSalt()
		if err != nil {
			return tx.Signed{}, fmt.Errorf("sign: %w", err)
		}
		t.Salt = salt
	}

	kryo := tx.KryoSerialize(t.Encode())

	sum256 := sha256.Sum256(kryo)
	sum256Hex := hex.EncodeToString(sum256[:])
	sum512 := sha512.Sum512([]byte(sum256Hex))

	sig := ecdsa.Sign(w.privateKey, sum512[:])

	return tx.Signed{
		Value: t,
		Proofs: []tx.Proof{
			{ID: w.PeerID(), Signature: hex.EncodeToString(sig.Serialize())},
		},
	}, nil
}

// mustHex decodes a hex string and panics if decoding fails.
func mustHex(s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return b
}
