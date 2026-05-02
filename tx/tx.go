package tx

// Ref identifies a previous transaction, holding the hash and the snapshot ordinal
// in which it was accepted. Every new transaction from an address must
// reference its latest Ref as its parent.
type Ref struct {
	Hash    string `json:"hash"`
	Ordinal uint64 `json:"ordinal"`
}

// Transfer is an unsigned value-transfer message. Construct one and hand it
// to wallet.Sign to produce a Signed ready for network.Send. Salt may be
// left zero — wallet.Sign will populate it with a random value before
// producing the signature.
type Transfer struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
	Amount      Amount `json:"amount"`
	Fee         Amount `json:"fee"`
	Parent      Ref    `json:"parent"`
	Salt        uint64 `json:"salt"`
}

// Signed wraps a Transfer with one or more Proofs authorizing it. This is
// the exact JSON shape L1 accepts at POST /transactions.
type Signed struct {
	Value  Transfer `json:"value"`
	Proofs []Proof  `json:"proofs"`
}

// Proof is one signature over a Transfer's canonical encoding.
type Proof struct {
	// ID is the signer's PeerID — uncompressed secp256k1 public key without
	// the leading 04 marker, as 128 hex characters.
	ID string `json:"id"`
	// Signature is a DER-encoded ECDSA signature, hex-encoded.
	Signature string `json:"signature"`
}
