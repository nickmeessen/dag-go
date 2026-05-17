// Package network provides an HTTP client for the Constellation (DAG)
// network. It has clients for both metagraphs and the global network.
// It exposes balance and last-tx-reference queries against L0,
// plus transaction submission and pending-tx polling against L1.
package network

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/nickmeessen/dag-go/tx"
)

// Client represents a DAG network client, used to interact with both metagraphs and the global network.
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

// Option is a function that configures a Client.
type Option func(*client)

// New creates a new Client with the given options.
func New(options ...Option) (Client, error) {
	c := &client{}
	for _, option := range options {
		option(c)
	}
	if c.httpClient == nil {
		c.httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	if c.l0URL == "" || c.l1URL == "" {
		return nil, ErrMissingNetworkConfiguration
	}
	c.balancePath = "/dag/"
	return c, nil
}

// NewMetagraphClient creates a new Client for the configured Metagraph network.
func NewMetagraphClient(l0URL, l1URL string, options ...Option) (Client, error) {
	if l0URL == "" || l1URL == "" {
		return nil, ErrMissingNetworkConfiguration
	}
	c := &client{}
	for _, option := range options {
		option(c)
	}
	if c.httpClient == nil {
		c.httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	c.l0URL = l0URL
	c.l1URL = l1URL
	c.balancePath = "/currency/"
	return c, nil
}

// WithHTTPClient sets the HTTP client to use for requests.
func WithHTTPClient(c *http.Client) Option {
	return func(n *client) {
		n.httpClient = c
	}
}

// WithMainNet configures the client to hit Constellation's production mainnet
// endpoints. DAG transferred here is real value, so use with care.
func WithMainNet() Option {
	return func(n *client) {
		n.l0URL = "https://l0-lb-mainnet.constellationnetwork.io"
		n.l1URL = "https://l1-lb-mainnet.constellationnetwork.io"
	}
}

// WithIntegrationNet configures the client to hit Constellation's IntegrationNet,
// the pre-production environment for testing. Tokens here have no real value
// and can be obtained from the faucet at
// https://faucet.constellationnetwork.io/integrationnet/faucet/<DAG_ADDRESS>.
func WithIntegrationNet() Option {
	return func(n *client) {
		n.l0URL = "https://l0-lb-integrationnet.constellationnetwork.io"
		n.l1URL = "https://l1-lb-integrationnet.constellationnetwork.io"
	}
}

func (c *client) doJSON(ctx context.Context, op, url, method string, body, out any) error {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("%s: marshal request body: %w", op, err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return fmt.Errorf("%s: build request: %w", op, err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrNodeUnreachable, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return ErrNotFound
	}
	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		return fmt.Errorf("%w: %s returned %d", ErrTxRejected, op, resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: unexpected status %d", op, resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("%s: decode response: %w", op, err)
	}
	return nil
}

// Balance returns the balance of the given DAG address on the configured
// network or metagraph. Returns ErrInvalidAddress for empty input, ErrNotFound if the
// address has no on-chain history, and ErrNodeUnreachable on transport-level
// failures.
func (c *client) Balance(ctx context.Context, address string) (tx.Amount, error) {
	if address == "" {
		return 0, ErrInvalidAddress
	}

	var r struct {
		Balance int64  `json:"balance"`
		Ordinal uint64 `json:"ordinal"`
	}
	url := c.l0URL + c.balancePath + address + "/balance"
	if err := c.doJSON(ctx, "balance", url, http.MethodGet, nil, &r); err != nil {
		return 0, err
	}
	return tx.Datoshi(r.Balance), nil
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
	if err := c.doJSON(ctx, "last-tx-ref", url, http.MethodGet, nil, &r); err != nil {
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
	if err := c.doJSON(ctx, "send", url, http.MethodPost, signed, &r); err != nil {
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
	if err := c.doJSON(ctx, "pending-tx", url, http.MethodGet, nil, &r); err != nil {
		return tx.Ref{}, err
	}
	return tx.Ref{Hash: r.Hash, Ordinal: r.Ordinal}, nil
}
