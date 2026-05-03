package network

import "github.com/go-errors/errors"

var (
	// ErrMissingNetworkConfiguration is returned when Client.New is called
	// without a network option (WithMainNet or WithIntegrationNet).
	ErrMissingNetworkConfiguration = errors.New("missing network configuration")

	// ErrInvalidAddress is returned when a supplied DAG address fails
	// client-side validation (empty, wrong format).
	ErrInvalidAddress = errors.New("invalid address")

	// ErrNotFound is returned when the remote returns HTTP 404,
	// a balance query against an address with no on-chain history,
	// or a pending-tx lookup for a tx that is no longer in the pool.
	ErrNotFound = errors.New("not found")

	// ErrNodeUnreachable is returned when the HTTP request fails at the
	// transport layer (DNS, TCP, TLS, timeout). Callers should treat this
	// as a retryable condition.
	ErrNodeUnreachable = errors.New("node unreachable")

	// ErrTxRejected is returned when L1 rejects a submitted transaction with a
	// 4xx status — typically malformed input, insufficient balance, bad parent
	// reference, or duplicate hash.
	ErrTxRejected = errors.New("transaction rejected")
)
