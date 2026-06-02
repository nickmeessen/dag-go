# dag-go 

[![CI](https://github.com/nickmeessen/dag-go/actions/workflows/ci.yml/badge.svg)](https://github.com/nickmeessen/dag-go/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/nickmeessen/dag-go.svg)](https://pkg.go.dev/github.com/nickmeessen/dag-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/nickmeessen/dag-go)](https://goreportcard.com/report/github.com/nickmeessen/dag-go)
[![Release](https://img.shields.io/github/v/release/nickmeessen/dag-go)](https://github.com/nickmeessen/dag-go/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE.md)

A Go SDK for interacting with the Constellation DAG network, mostly ported from the JavaScript SDK https://github.com/StardustCollective/dag4.js


## Usage

```go
ctx := context.Background()

sender, _ := wallet.FromPrivateKey(os.Getenv("PRIVATE_KEY"))
recipient, _ := wallet.New()

client, _ := network.New(network.WithIntegrationNet())

bal, _ := client.Balance(ctx, sender.Address())
ref, _ := client.LastTxRef(ctx, sender.Address())

t := tx.Transfer{
    Source:      sender.Address(),
    Destination: recipient.Address(),
    Amount:      tx.Token(8, 0.001),
    Fee:         tx.Token(8, 0.0001),
    Parent:      ref,
}

signed, _ := sender.Sign(t)
hash, _ := client.Send(ctx, signed)
fmt.Printf("submitted: %s (balance was %s DAG)\n", hash, bal.FormatToken(8))
```

See [`examples/basic`](examples/basic/main.go) for the full demo including pending-tx polling.

For metagraph token transfers, use `network.NewMetagraphClient` with the metagraph's L0/L1 URLs, see [`examples/metagraph`](examples/metagraph/main.go).

### Reading confirmed state

```go
be, _ := network.NewBlockExplorer(network.WithMainNet())

bal, _ := be.AddressBalance(ctx, addr)
fmt.Printf("balance: %s (at ordinal %d)\n", bal.Balance.FormatToken(8), bal.Ordinal)

txs, _, _ := be.TransactionsByAddress(ctx, addr, network.PageOpts{Limit: 10})
locks, _, _ := be.TokenLocksByAddress(ctx, addr, network.PageOpts{ActiveOnly: true})
```

For metagraph token reads, swap `NewBlockExplorer` for `NewMetagraphBlockExplorer(metagraphID, ...)`. See [`examples/blockexplorer`](examples/blockexplorer/main.go).

## Installation

```bash
go get github.com/nickmeessen/dag-go
```

## Disclaimer

This is an unofficial third-party SDK. Not affiliated with or endorsed by
Constellation Network or Stardust Collective. Use at your own risk.

## License

See [LICENSE.md](LICENSE.md)

## Implementation Notes

The transaction canonical encoding, Kryo serialization, and signing chain in `tx/encoding.go` and `wallet.Sign` are ports of [dag4.js][]'s `dag4-keystore/src/transaction-v2.ts`, `tx-encode.ts`, and `key-store.ts`.

The crypto parts were ported with LLM assistance, the reference has several non-obvious
quirks (a custom length-prefixed concatenation that isn't JSON, a Kryo wrap with a quirky Uint16Array varint, a double-hash chain that goes SHA-256 → hex-encode → SHA-512 before ECDSA) that are faster to translate than to re-derive.

Correctness is verified by cross-validation tests in `wallet/wallet_test.go`: a fixed-input transfer signed in both dag4.js and `dag-go` must produce the exact same DER-encoded signature. The suite currently asserts byte-parity against a vector generated from dag4.js v2.8.1.

Comments on the crypto/encoding code (tx/encoding.go and wallet.Sign) were LLM-assisted to explain the ported logic and dag4.js quirks. Other comments and design decisions throughout the SDK are mine.

[dag4.js]: https://github.com/StardustCollective/dag4.js

