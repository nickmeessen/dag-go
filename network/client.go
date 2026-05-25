// Package network provides HTTP clients for the Constellation network.
// Client handles balance, last-tx-ref, send, and pending-tx against L0/L1 for
// either DAG or a metagraph. BlockExplorer reads confirmed state (transactions,
// balances, locks, allow-spends) from the Block Explorer.
package network

import (
	"context"
	"net/http"

	"github.com/nickmeessen/dag-go/tx"
)

// Client interacts with either the global DAG network or a metagraph.
type Client interface {
	// Balance retrieves the current balance for the given address from L0.
	Balance(ctx context.Context, address string) (tx.Amount, error)
	// LastTxRef returns the most recent accepted transaction reference for the given address.
	LastTxRef(ctx context.Context, address string) (tx.Ref, error)
	// Send submits a signed transaction to L1's transaction pool.
	Send(ctx context.Context, signed tx.Signed) (string, error)
	// PendingTx looks up a transaction by hash in L1's pending pool.
	PendingTx(ctx context.Context, hash string) (tx.Ref, error)
}

type client struct {
	httpClient  *http.Client
	l0URL       string
	l1URL       string
	balancePath string
}

// New creates a new Client with the given options.
func New(options ...Option) (Client, error) {
	cfg := newConfig(options...)
	if cfg.l0URL == "" || cfg.l1URL == "" {
		return nil, ErrMissingNetworkConfiguration
	}
	return &client{
		httpClient:  cfg.httpClient,
		l0URL:       cfg.l0URL,
		l1URL:       cfg.l1URL,
		balancePath: "/dag/",
	}, nil
}

// NewMetagraphClient creates a new Client for the configured Metagraph network.
func NewMetagraphClient(l0URL, l1URL string, options ...Option) (Client, error) {
	if l0URL == "" || l1URL == "" {
		return nil, ErrMissingNetworkConfiguration
	}
	cfg := newConfig(options...)
	return &client{
		httpClient:  cfg.httpClient,
		l0URL:       l0URL,
		l1URL:       l1URL,
		balancePath: "/currency/",
	}, nil
}

// Balance returns the balance for the given address on the configured
// network or metagraph. Returns ErrInvalidAddress for empty input,
// ErrNotFound if the address has no on-chain history.
func (c *client) Balance(ctx context.Context, address string) (tx.Amount, error) {
	if address == "" {
		return 0, ErrInvalidAddress
	}

	var r struct {
		Balance int64  `json:"balance"`
		Ordinal uint64 `json:"ordinal"`
	}
	url := c.l0URL + c.balancePath + address + "/balance"
	if err := doJSON(ctx, c.httpClient, "balance", url, http.MethodGet, nil, &r); err != nil {
		return 0, err
	}
	return tx.Datum(r.Balance), nil
}

// LastTxRef returns the most recent accepted transaction reference for the
// given address. For addresses that have never sent, the returned Ref has
// Ordinal 0 and an all-zero Hash, which is the valid parent for a first
// transaction from that address.
func (c *client) LastTxRef(ctx context.Context, address string) (tx.Ref, error) {
	if address == "" {
		return tx.Ref{}, ErrInvalidAddress
	}

	var r struct {
		Hash    string `json:"hash"`
		Ordinal uint64 `json:"ordinal"`
	}
	url := c.l1URL + "/transactions/last-reference/" + address
	if err := doJSON(ctx, c.httpClient, "last-tx-ref", url, http.MethodGet, nil, &r); err != nil {
		return tx.Ref{}, err
	}
	return tx.Ref{Hash: r.Hash, Ordinal: r.Ordinal}, nil
}

// Send submits a signed transaction to L1's transaction pool and returns the
// assigned transaction hash on success. Returns ErrNodeUnreachable on
// transport-level failures.
func (c *client) Send(ctx context.Context, signed tx.Signed) (string, error) {
	var r struct {
		Hash string `json:"hash"`
	}

	url := c.l1URL + "/transactions"
	if err := doJSON(ctx, c.httpClient, "send", url, http.MethodPost, signed, &r); err != nil {
		return "", err
	}
	return r.Hash, nil
}

// PendingTx looks up a transaction by hash in L1's pending pool. Returns
// ErrNotFound when the transaction is no longer pending — typically because
// it has been included in a snapshot or rejected. Callers can use
// ErrNotFound as a (loose) confirmation signal in polling loops.
func (c *client) PendingTx(ctx context.Context, hash string) (tx.Ref, error) {
	var r struct {
		Hash    string `json:"hash"`
		Ordinal uint64 `json:"ordinal"`
	}
	url := c.l1URL + "/transactions/pending/" + hash
	if err := doJSON(ctx, c.httpClient, "pending-tx", url, http.MethodGet, nil, &r); err != nil {
		return tx.Ref{}, err
	}
	return tx.Ref{Hash: r.Hash, Ordinal: r.Ordinal}, nil
}
