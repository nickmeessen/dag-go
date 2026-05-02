// Package tx defines transaction value types (Transfer, Signed, Ref, Proof,
// Amount), the canonical encoding used as the signing pre-image, the Kryo
// serialization wrapper, and salt generation for the Constellation (DAG)
// network.
//
// The encoding, Kryo serialization, and salt generator in tx/encoding.go are
// ports of dag4.js's transaction-v2.ts and tx-encode.ts. Cross-validation
// tests in wallet/wallet_test.go assert byte parity against dag4.js v2.8.1.
package tx

import (
	cryptorand "crypto/rand"
	"fmt"
	"strconv"
)

// minSalt enforces minimum hash complexity on V2 transactions:
// Number.MAX_SAFE_INTEGER - 2^48.
const minSalt = uint64(8725724278030335)

// Encode returns the length-prefixed concatenation used as the signing
// pre-image for a V2 transaction. Every length is an ASCII decimal string
// (not a binary integer), and the output is plain ASCII — feed through
// KryoSerialize before hashing.
func (t Transfer) Encode() string {
	amountHex := strconv.FormatInt(int64(t.Amount), 16)
	ordinalStr := strconv.FormatUint(t.Parent.Ordinal, 10)
	feeStr := strconv.FormatInt(int64(t.Fee), 10)
	saltHex := strconv.FormatUint(t.Salt, 16)

	return "2" +
		strconv.Itoa(len(t.Source)) + t.Source +
		strconv.Itoa(len(t.Destination)) + t.Destination +
		strconv.Itoa(len(amountHex)) + amountHex +
		strconv.Itoa(len(t.Parent.Hash)) + t.Parent.Hash +
		strconv.Itoa(len(ordinalStr)) + ordinalStr +
		strconv.Itoa(len(feeStr)) + feeStr +
		strconv.Itoa(len(saltHex)) + saltHex
}

// KryoSerialize wraps an encoded V2 transfer with a 0x03 type byte and a
// variable-length length prefix, returning raw bytes ready to be hashed.
func KryoSerialize(msg string) []byte {
	prefix := []byte{0x03}
	prefix = append(prefix, utf8Length(len(msg)+1)...)
	return append(prefix, []byte(msg)...)
}

// GenerateSalt returns a random salt in [minSalt, minSalt+2^48). The floor
// enforces minimum hash complexity required by the protocol.
func GenerateSalt() (uint64, error) {
	var b [6]byte
	if _, err := cryptorand.Read(b[:]); err != nil {
		return 0, fmt.Errorf("generate salt: %w", err)
	}
	var r uint64
	for _, c := range b {
		r = (r << 8) | uint64(c)
	}
	return minSalt + r, nil
}

// utf8Length is Kryo's variable-length integer encoding. Each logical byte
// is emitted once — the JS reference appears to emit two bytes per value
// via Uint16Array, but Buffer.from(Uint16Array) truncates each slot to a
// single byte, so this one-byte-per-slot implementation matches.
func utf8Length(value int) []byte {
	switch {
	case value>>6 == 0:
		return []byte{byte(value | 0x80)}
	case value>>13 == 0:
		return []byte{
			byte(value | 0x40 | 0x80),
			byte(value >> 6),
		}
	case value>>20 == 0:
		return []byte{
			byte(value | 0x40 | 0x80),
			byte((value >> 6) | 0x80),
			byte(value >> 13),
		}
	case value>>27 == 0:
		return []byte{
			byte(value | 0x40 | 0x80),
			byte((value >> 6) | 0x80),
			byte((value >> 13) | 0x80),
			byte(value >> 20),
		}
	default:
		return []byte{
			byte(value | 0x40 | 0x80),
			byte((value >> 6) | 0x80),
			byte((value >> 13) | 0x80),
			byte((value >> 20) | 0x80),
			byte(value >> 27),
		}
	}
}
