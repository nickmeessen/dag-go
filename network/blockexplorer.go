package network

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/nickmeessen/dag-go/tx"
)

// BlockExplorer reads confirmed on-chain state from a Constellation Block
// Explorer. Use NewBlockExplorer for DAG and NewMetagraphBlockExplorer for a
// metagraph's currency token.
type BlockExplorer interface {
	// Transaction returns a confirmed transaction by hash.
	Transaction(ctx context.Context, hash string) (ConfirmedTransaction, error)
	// TransactionsByAddress returns confirmed transactions for the address, most recent first.
	TransactionsByAddress(ctx context.Context, address string, opts PageOpts) (txs []ConfirmedTransaction, next string, err error)
	// AddressBalance returns the address's available balance and the snapshot ordinal it's current at.
	AddressBalance(ctx context.Context, address string) (AddressBalance, error)
	// TokenLocksByAddress returns the address's token locks. Set PageOpts.ActiveOnly to skip released ones.
	TokenLocksByAddress(ctx context.Context, address string, opts PageOpts) (locks []TokenLock, next string, err error)
	// AllowSpendsByAddress returns the address's allow-spends. Set PageOpts.ActiveOnly to skip consumed/expired ones.
	AllowSpendsByAddress(ctx context.Context, address string, opts PageOpts) (spends []AllowSpend, next string, err error)
}

type blockExplorer struct {
	httpClient *http.Client
	beURL      string
	pathPrefix string // "" for DAG, "/currency/{metagraphID}" for metagraph
}

// ConfirmedTransaction is a finalized transaction returned by the Block
// Explorer. On currency txs the Snapshot* fields refer to the currency
// snapshot; GlobalSnapshot* to the global snapshot that included it.
type ConfirmedTransaction struct {
	Hash                  string    `json:"hash"`
	Ordinal               uint64    `json:"ordinal"`
	Source                string    `json:"source"`
	Destination           string    `json:"destination"`
	Amount                tx.Amount `json:"amount"`
	Fee                   tx.Amount `json:"fee"`
	Parent                tx.Ref    `json:"parent"`
	Salt                  uint64    `json:"salt"`
	BlockHash             string    `json:"blockHash"`
	SnapshotHash          string    `json:"snapshotHash"`
	SnapshotOrdinal       uint64    `json:"snapshotOrdinal"`
	GlobalSnapshotHash    string    `json:"globalSnapshotHash"`
	GlobalSnapshotOrdinal uint64    `json:"globalSnapshotOrdinal"`
	Timestamp             time.Time `json:"timestamp"`
}

// AddressBalance is the available balance for an address, as of the given
// snapshot ordinal. Locked balance lives separately under TokenLocksByAddress.
type AddressBalance struct {
	Address string    `json:"address"`
	Balance tx.Amount `json:"balance"`
	Ordinal uint64    `json:"ordinal"`
}

// TokenLock is a token-lock record.
type TokenLock struct {
	Hash                  string    `json:"hash"`
	Ordinal               uint64    `json:"ordinal"`
	Source                string    `json:"source"`
	Amount                tx.Amount `json:"amount"`
	ParentHash            string    `json:"parentHash"`
	UnlockEpoch           *uint64   `json:"unlockEpoch"`       // nil if no scheduled unlock
	UnlockedAtOrdinal     *uint64   `json:"unlockedAtOrdinal"` // nil while the lock is active
	GlobalSnapshotHash    string    `json:"globalSnapshotHash"`
	GlobalSnapshotOrdinal uint64    `json:"globalSnapshotOrdinal"`
	Timestamp             time.Time `json:"timestamp"`
}

// Active reports whether the lock is still in effect (not yet released).
func (l TokenLock) Active() bool {
	return l.UnlockedAtOrdinal == nil
}

// AllowSpend is tokens earmarked for a future spend to Destination, valid
// until LastValidEpochProgress. Whether a spend has been consumed or expired
// isn't visible client-side; use PageOpts.ActiveOnly to filter server-side.
type AllowSpend struct {
	Hash                   string    `json:"hash"`
	Ordinal                uint64    `json:"ordinal"`
	Source                 string    `json:"source"`
	Destination            string    `json:"destination"`
	Amount                 tx.Amount `json:"amount"`
	Fee                    tx.Amount `json:"fee"`
	LastValidEpochProgress uint64    `json:"lastValidEpochProgress"`
	SnapshotHash           string    `json:"snapshotHash"`
	GlobalSnapshotHash     string    `json:"globalSnapshotHash"`
	GlobalSnapshotOrdinal  uint64    `json:"globalSnapshotOrdinal"`
	Timestamp              time.Time `json:"timestamp"`
}

// PageOpts controls cursor-based pagination on list endpoints.
type PageOpts struct {
	Limit        int
	SearchAfter  string // mutually exclusive with SearchBefore and Next
	SearchBefore string
	Next         string // opaque cursor from the previous response's meta.next
	ActiveOnly   bool   // applies only to TokenLocksByAddress and AllowSpendsByAddress
}

func (p PageOpts) query() string {
	v := url.Values{}
	if p.Limit > 0 {
		v.Set("limit", strconv.Itoa(p.Limit))
	}
	if p.SearchAfter != "" {
		v.Set("search_after", p.SearchAfter)
	}
	if p.SearchBefore != "" {
		v.Set("search_before", p.SearchBefore)
	}
	if p.Next != "" {
		v.Set("next", p.Next)
	}
	if p.ActiveOnly {
		v.Set("active", "true")
	}
	return v.Encode()
}

// NewBlockExplorer creates a Block Explorer client for DAG reads. Requires
// WithMainNet() or WithIntegrationNet().
func NewBlockExplorer(options ...Option) (BlockExplorer, error) {
	cfg := newConfig(options...)
	if cfg.beURL == "" {
		return nil, ErrMissingNetworkConfiguration
	}
	return &blockExplorer{
		httpClient: cfg.httpClient,
		beURL:      cfg.beURL,
	}, nil
}

// NewMetagraphBlockExplorer creates a Block Explorer client scoped to a
// metagraph's currency token; reads route under /currency/{metagraphID}.
// Requires WithMainNet() or WithIntegrationNet().
func NewMetagraphBlockExplorer(metagraphID string, options ...Option) (BlockExplorer, error) {
	if metagraphID == "" {
		return nil, ErrMissingNetworkConfiguration
	}
	cfg := newConfig(options...)
	if cfg.beURL == "" {
		return nil, ErrMissingNetworkConfiguration
	}
	return &blockExplorer{
		httpClient: cfg.httpClient,
		beURL:      cfg.beURL,
		pathPrefix: "/currency/" + metagraphID,
	}, nil
}

// Transaction returns a confirmed transaction by hash. Returns ErrNotFound if
// no transaction with that hash exists.
func (b *blockExplorer) Transaction(ctx context.Context, hash string) (ConfirmedTransaction, error) {
	var r struct {
		Data ConfirmedTransaction `json:"data"`
	}
	u := b.beURL + b.pathPrefix + "/transactions/" + hash
	if err := doJSON(ctx, b.httpClient, "transaction", u, http.MethodGet, nil, &r); err != nil {
		return ConfirmedTransaction{}, err
	}
	return r.Data, nil
}

// TransactionsByAddress returns confirmed transactions for the address, most
// recent first. The returned next cursor, if non-empty, can be passed back as
// PageOpts.Next to fetch the next page. Returns ErrInvalidAddress on empty
// input.
func (b *blockExplorer) TransactionsByAddress(ctx context.Context, address string, opts PageOpts) ([]ConfirmedTransaction, string, error) {
	if address == "" {
		return nil, "", ErrInvalidAddress
	}
	var r struct {
		Data []ConfirmedTransaction `json:"data"`
		Meta struct {
			Next string `json:"next"`
		} `json:"meta"`
	}
	u := b.beURL + b.pathPrefix + "/addresses/" + address + "/transactions"
	if q := opts.query(); q != "" {
		u += "?" + q
	}
	if err := doJSON(ctx, b.httpClient, "transactions-by-address", u, http.MethodGet, nil, &r); err != nil {
		return nil, "", err
	}
	return r.Data, r.Meta.Next, nil
}

// AddressBalance returns the address's available balance as of a specific
// snapshot ordinal. Returns ErrInvalidAddress on empty input.
func (b *blockExplorer) AddressBalance(ctx context.Context, address string) (AddressBalance, error) {
	if address == "" {
		return AddressBalance{}, ErrInvalidAddress
	}
	var r struct {
		Data AddressBalance `json:"data"`
	}
	u := b.beURL + b.pathPrefix + "/addresses/" + address + "/balance"
	if err := doJSON(ctx, b.httpClient, "address-balance", u, http.MethodGet, nil, &r); err != nil {
		return AddressBalance{}, err
	}
	return r.Data, nil
}

// TokenLocksByAddress returns the address's token locks. Set
// PageOpts.ActiveOnly to skip released locks server-side. Returns
// ErrInvalidAddress on empty input.
func (b *blockExplorer) TokenLocksByAddress(ctx context.Context, address string, opts PageOpts) ([]TokenLock, string, error) {
	if address == "" {
		return nil, "", ErrInvalidAddress
	}
	var r struct {
		Data []TokenLock `json:"data"`
		Meta struct {
			Next string `json:"next"`
		} `json:"meta"`
	}
	u := b.beURL + b.pathPrefix + "/addresses/" + address + "/token-locks"
	if q := opts.query(); q != "" {
		u += "?" + q
	}
	if err := doJSON(ctx, b.httpClient, "token-locks-by-address", u, http.MethodGet, nil, &r); err != nil {
		return nil, "", err
	}
	return r.Data, r.Meta.Next, nil
}

// AllowSpendsByAddress returns the address's allow-spends. Set
// PageOpts.ActiveOnly to skip consumed/expired ones server-side. Returns
// ErrInvalidAddress on empty input.
func (b *blockExplorer) AllowSpendsByAddress(ctx context.Context, address string, opts PageOpts) ([]AllowSpend, string, error) {
	if address == "" {
		return nil, "", ErrInvalidAddress
	}
	var r struct {
		Data []AllowSpend `json:"data"`
		Meta struct {
			Next string `json:"next"`
		} `json:"meta"`
	}
	u := b.beURL + b.pathPrefix + "/addresses/" + address + "/allow-spends"
	if q := opts.query(); q != "" {
		u += "?" + q
	}
	if err := doJSON(ctx, b.httpClient, "allow-spends-by-address", u, http.MethodGet, nil, &r); err != nil {
		return nil, "", err
	}
	return r.Data, r.Meta.Next, nil
}
