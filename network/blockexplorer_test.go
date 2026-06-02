package network

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/nickmeessen/dag-go/tx"
	"github.com/stretchr/testify/suite"
)

type BlockExplorerTestSuite struct {
	suite.Suite
}

func (s *BlockExplorerTestSuite) TestNewBlockExplorer() {
	s.Run("routes transaction lookup to mainnet BE", func() {
		var got string
		be, err := NewBlockExplorer(WithMainNet(), WithHTTPClient(captureURL(&got)))
		s.Require().NoError(err)

		_, _ = be.Transaction(context.Background(), "abc123")
		s.Contains(got, "https://be-mainnet.constellationnetwork.io/transactions/abc123")
	})

	s.Run("routes transaction lookup to integrationNet BE", func() {
		var got string
		be, err := NewBlockExplorer(WithIntegrationNet(), WithHTTPClient(captureURL(&got)))
		s.Require().NoError(err)

		_, _ = be.Transaction(context.Background(), "abc123")
		s.Contains(got, "https://be-integrationnet.constellationnetwork.io/transactions/abc123")
	})

	s.Run("fails with missing network config", func() {
		_, err := NewBlockExplorer()
		s.ErrorIs(err, ErrMissingNetworkConfiguration)
	})

	s.Run("uses custom HTTP client when provided", func() {
		called := false
		rt := transportFunc(func(_ *http.Request) (*http.Response, error) {
			called = true
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(`{"data":{}}`)),
			}, nil
		})
		be, err := NewBlockExplorer(WithMainNet(), WithHTTPClient(&http.Client{Transport: rt}))
		s.Require().NoError(err)

		_, _ = be.Transaction(context.Background(), "abc")
		s.True(called)
	})
}

func (s *BlockExplorerTestSuite) TestNewMetagraphBlockExplorer() {
	const metagraphID = "DAG7ChnhUF7uKgn8tXy45aj4zn9AFuhaZr8VXY43"
	const addr = "DAG3jifKUZPc213rRLSfZVSLZfPRfX7fwTGh8tsy"

	s.Run("routes Transaction under /currency/{id}", func() {
		var got string
		be, err := NewMetagraphBlockExplorer(metagraphID, WithMainNet(), WithHTTPClient(captureURL(&got)))
		s.Require().NoError(err)

		_, _ = be.Transaction(context.Background(), "abc")
		s.Contains(got, "https://be-mainnet.constellationnetwork.io/currency/"+metagraphID+"/transactions/abc")
	})

	s.Run("routes TransactionsByAddress under /currency/{id}", func() {
		var got string
		be, err := NewMetagraphBlockExplorer(metagraphID, WithMainNet(), WithHTTPClient(captureURL(&got)))
		s.Require().NoError(err)

		_, _, _ = be.TransactionsByAddress(context.Background(), addr, PageOpts{})
		s.Contains(got, "https://be-mainnet.constellationnetwork.io/currency/"+metagraphID+"/addresses/"+addr+"/transactions")
	})

	s.Run("routes AddressBalance under /currency/{id}", func() {
		var got string
		be, err := NewMetagraphBlockExplorer(metagraphID, WithMainNet(), WithHTTPClient(captureURL(&got)))
		s.Require().NoError(err)

		_, _ = be.AddressBalance(context.Background(), addr)
		s.Equal("https://be-mainnet.constellationnetwork.io/currency/"+metagraphID+"/addresses/"+addr+"/balance", got)
	})

	s.Run("routes TokenLocksByAddress under /currency/{id}", func() {
		var got string
		be, err := NewMetagraphBlockExplorer(metagraphID, WithMainNet(), WithHTTPClient(captureURL(&got)))
		s.Require().NoError(err)

		_, _, _ = be.TokenLocksByAddress(context.Background(), addr, PageOpts{})
		s.Contains(got, "https://be-mainnet.constellationnetwork.io/currency/"+metagraphID+"/addresses/"+addr+"/token-locks")
	})

	s.Run("routes AllowSpendsByAddress under /currency/{id}", func() {
		var got string
		be, err := NewMetagraphBlockExplorer(metagraphID, WithMainNet(), WithHTTPClient(captureURL(&got)))
		s.Require().NoError(err)

		_, _, _ = be.AllowSpendsByAddress(context.Background(), addr, PageOpts{})
		s.Contains(got, "https://be-mainnet.constellationnetwork.io/currency/"+metagraphID+"/addresses/"+addr+"/allow-spends")
	})

	s.Run("fails with empty metagraphID", func() {
		_, err := NewMetagraphBlockExplorer("", WithMainNet())
		s.ErrorIs(err, ErrMissingNetworkConfiguration)
	})

	s.Run("fails with missing network config", func() {
		_, err := NewMetagraphBlockExplorer(metagraphID)
		s.ErrorIs(err, ErrMissingNetworkConfiguration)
	})
}

func (s *BlockExplorerTestSuite) TestTransaction() {
	s.Run("decodes confirmed transaction", func() {
		body := `{"data":{
			"hash":"abc",
			"ordinal":42,
			"source":"DAG3jifKUZPc213rRLSfZVSLZfPRfX7fwTGh8tsy",
			"destination":"DAG58AYZbPHyiYQcmD96T3zWK2HyN3Zc3cjWGqi7",
			"amount":100000,
			"fee":1000,
			"parent":{"hash":"parenthash","ordinal":41},
			"salt":12345,
			"blockHash":"blockhash",
			"snapshotHash":"snaphash",
			"snapshotOrdinal":100,
			"globalSnapshotHash":"gsnaphash",
			"globalSnapshotOrdinal":200,
			"timestamp":"2026-05-20T12:34:56Z"
		}}`
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, body)
		}))
		defer srv.Close()

		be := &blockExplorer{httpClient: srv.Client(), beURL: srv.URL}

		got, err := be.Transaction(context.Background(), "abc")
		s.Require().NoError(err)
		s.Equal("abc", got.Hash)
		s.Equal(uint64(42), got.Ordinal)
		s.Equal("DAG3jifKUZPc213rRLSfZVSLZfPRfX7fwTGh8tsy", got.Source)
		s.Equal("DAG58AYZbPHyiYQcmD96T3zWK2HyN3Zc3cjWGqi7", got.Destination)
		s.Equal(tx.Amount(100000), got.Amount)
		s.Equal(tx.Amount(1000), got.Fee)
		s.Equal("parenthash", got.Parent.Hash)
		s.Equal(uint64(41), got.Parent.Ordinal)
		s.Equal(uint64(12345), got.Salt)
		s.Equal("blockhash", got.BlockHash)
		s.Equal("snaphash", got.SnapshotHash)
		s.Equal(uint64(100), got.SnapshotOrdinal)
		s.Equal("gsnaphash", got.GlobalSnapshotHash)
		s.Equal(uint64(200), got.GlobalSnapshotOrdinal)
		s.Equal(time.Date(2026, 5, 20, 12, 34, 56, 0, time.UTC), got.Timestamp)
	})

	s.Run("returns ErrNotFound on 404", func() {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer srv.Close()

		be := &blockExplorer{httpClient: srv.Client(), beURL: srv.URL}
		_, err := be.Transaction(context.Background(), "missing")
		s.ErrorIs(err, ErrNotFound)
	})
}

func (s *BlockExplorerTestSuite) TestTransactionsByAddress() {
	s.Run("decodes paged transactions and next cursor", func() {
		body := `{
			"data":[
				{"hash":"tx1","ordinal":1,"amount":100,"fee":0,"timestamp":"2026-05-20T10:00:00Z"},
				{"hash":"tx2","ordinal":2,"amount":200,"fee":0,"timestamp":"2026-05-20T11:00:00Z"}
			],
			"meta":{"next":"cursor-abc"}
		}`
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, body)
		}))
		defer srv.Close()

		be := &blockExplorer{httpClient: srv.Client(), beURL: srv.URL}
		txs, next, err := be.TransactionsByAddress(context.Background(), "DAG3jifKUZPc213rRLSfZVSLZfPRfX7fwTGh8tsy", PageOpts{})
		s.Require().NoError(err)
		s.Len(txs, 2)
		s.Equal("tx1", txs[0].Hash)
		s.Equal("tx2", txs[1].Hash)
		s.Equal("cursor-abc", next)
	})

	s.Run("passes pagination query params", func() {
		var capturedQuery string
		rt := transportFunc(func(r *http.Request) (*http.Response, error) {
			capturedQuery = r.URL.RawQuery
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(`{"data":[],"meta":{"next":""}}`)),
			}, nil
		})
		be := &blockExplorer{httpClient: &http.Client{Transport: rt}, beURL: "http://test-be"}

		_, _, err := be.TransactionsByAddress(context.Background(), "DAG3jifKUZPc213rRLSfZVSLZfPRfX7fwTGh8tsy", PageOpts{
			Limit:        25,
			SearchAfter:  "after-hash",
			SearchBefore: "before-hash",
			Next:         "next-token",
		})
		s.Require().NoError(err)
		s.Contains(capturedQuery, "limit=25")
		s.Contains(capturedQuery, "search_after=after-hash")
		s.Contains(capturedQuery, "search_before=before-hash")
		s.Contains(capturedQuery, "next=next-token")
	})

	s.Run("omits empty pagination params", func() {
		var capturedURL string
		rt := transportFunc(func(r *http.Request) (*http.Response, error) {
			capturedURL = r.URL.String()
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(`{"data":[],"meta":{"next":""}}`)),
			}, nil
		})
		be := &blockExplorer{httpClient: &http.Client{Transport: rt}, beURL: "http://test-be"}

		_, _, _ = be.TransactionsByAddress(context.Background(), "DAG3jifKUZPc213rRLSfZVSLZfPRfX7fwTGh8tsy", PageOpts{})
		s.NotContains(capturedURL, "?")
	})

	s.Run("returns ErrInvalidAddress on empty address", func() {
		be := &blockExplorer{httpClient: http.DefaultClient, beURL: "http://test-be"}
		_, _, err := be.TransactionsByAddress(context.Background(), "", PageOpts{})
		s.ErrorIs(err, ErrInvalidAddress)
	})
}

func (s *BlockExplorerTestSuite) TestAddressBalance() {
	s.Run("decodes balance with ordinal", func() {
		body := `{"data":{"address":"DAG3jifKUZPc213rRLSfZVSLZfPRfX7fwTGh8tsy","balance":250000,"ordinal":777}}`
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, body)
		}))
		defer srv.Close()

		be := &blockExplorer{httpClient: srv.Client(), beURL: srv.URL}
		got, err := be.AddressBalance(context.Background(), "DAG3jifKUZPc213rRLSfZVSLZfPRfX7fwTGh8tsy")
		s.Require().NoError(err)
		s.Equal("DAG3jifKUZPc213rRLSfZVSLZfPRfX7fwTGh8tsy", got.Address)
		s.Equal(tx.Amount(250000), got.Balance)
		s.Equal(uint64(777), got.Ordinal)
	})

	s.Run("routes to /addresses/{addr}/balance", func() {
		var capturedURL string
		rt := transportFunc(func(r *http.Request) (*http.Response, error) {
			capturedURL = r.URL.String()
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"data":{}}`))}, nil
		})
		be := &blockExplorer{httpClient: &http.Client{Transport: rt}, beURL: "http://test-be"}

		_, _ = be.AddressBalance(context.Background(), "DAG3jifKUZPc213rRLSfZVSLZfPRfX7fwTGh8tsy")
		s.Equal("http://test-be/addresses/DAG3jifKUZPc213rRLSfZVSLZfPRfX7fwTGh8tsy/balance", capturedURL)
	})

	s.Run("returns ErrInvalidAddress on empty address", func() {
		be := &blockExplorer{httpClient: http.DefaultClient, beURL: "http://test-be"}
		_, err := be.AddressBalance(context.Background(), "")
		s.ErrorIs(err, ErrInvalidAddress)
	})
}

func (s *BlockExplorerTestSuite) TestTokenLocksByAddress() {
	const addr = "DAG3jifKUZPc213rRLSfZVSLZfPRfX7fwTGh8tsy"

	s.Run("decodes locks with active and released states", func() {
		body := `{
			"data":[
				{"hash":"lock1","ordinal":1,"source":"` + addr + `","amount":1000,"currencyId":"","parentHash":"p1","unlockEpoch":null,"unlockedAtOrdinal":null,"globalSnapshotHash":"g1","globalSnapshotOrdinal":100,"timestamp":"2026-05-20T10:00:00Z"},
				{"hash":"lock2","ordinal":2,"source":"` + addr + `","amount":2000,"currencyId":"","parentHash":"p2","unlockEpoch":500,"unlockedAtOrdinal":150,"globalSnapshotHash":"g2","globalSnapshotOrdinal":200,"timestamp":"2026-05-20T11:00:00Z"}
			],
			"meta":{"next":""}
		}`
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, body)
		}))
		defer srv.Close()

		be := &blockExplorer{httpClient: srv.Client(), beURL: srv.URL}
		locks, _, err := be.TokenLocksByAddress(context.Background(), addr, PageOpts{})
		s.Require().NoError(err)
		s.Require().Len(locks, 2)

		s.Equal("lock1", locks[0].Hash)
		s.Equal(tx.Amount(1000), locks[0].Amount)
		s.Nil(locks[0].UnlockedAtOrdinal)
		s.True(locks[0].Active())

		s.Equal("lock2", locks[1].Hash)
		s.NotNil(locks[1].UnlockedAtOrdinal)
		s.Equal(uint64(150), *locks[1].UnlockedAtOrdinal)
		s.NotNil(locks[1].UnlockEpoch)
		s.Equal(uint64(500), *locks[1].UnlockEpoch)
		s.False(locks[1].Active())
	})

	s.Run("sends active=true when ActiveOnly is set", func() {
		var capturedQuery string
		rt := transportFunc(func(r *http.Request) (*http.Response, error) {
			capturedQuery = r.URL.RawQuery
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"data":[],"meta":{"next":""}}`))}, nil
		})
		be := &blockExplorer{httpClient: &http.Client{Transport: rt}, beURL: "http://test-be"}

		_, _, _ = be.TokenLocksByAddress(context.Background(), addr, PageOpts{ActiveOnly: true})
		s.Contains(capturedQuery, "active=true")
	})

	s.Run("omits active when ActiveOnly is false", func() {
		var capturedURL string
		rt := transportFunc(func(r *http.Request) (*http.Response, error) {
			capturedURL = r.URL.String()
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"data":[],"meta":{"next":""}}`))}, nil
		})
		be := &blockExplorer{httpClient: &http.Client{Transport: rt}, beURL: "http://test-be"}

		_, _, _ = be.TokenLocksByAddress(context.Background(), addr, PageOpts{})
		s.NotContains(capturedURL, "active=")
	})

	s.Run("returns ErrInvalidAddress on empty address", func() {
		be := &blockExplorer{httpClient: http.DefaultClient, beURL: "http://test-be"}
		_, _, err := be.TokenLocksByAddress(context.Background(), "", PageOpts{})
		s.ErrorIs(err, ErrInvalidAddress)
	})
}

func (s *BlockExplorerTestSuite) TestAllowSpendsByAddress() {
	const addr = "DAG3jifKUZPc213rRLSfZVSLZfPRfX7fwTGh8tsy"

	s.Run("decodes allow-spends", func() {
		body := `{
			"data":[
				{"hash":"as1","ordinal":1,"source":"` + addr + `","destination":"DAG58AYZbPHyiYQcmD96T3zWK2HyN3Zc3cjWGqi7","amount":500,"fee":10,"currencyId":"","lastValidEpochProgress":1000,"snapshotHash":"s1","globalSnapshotHash":"g1","globalSnapshotOrdinal":50,"timestamp":"2026-05-20T10:00:00Z"}
			],
			"meta":{"next":""}
		}`
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, body)
		}))
		defer srv.Close()

		be := &blockExplorer{httpClient: srv.Client(), beURL: srv.URL}
		spends, _, err := be.AllowSpendsByAddress(context.Background(), addr, PageOpts{})
		s.Require().NoError(err)
		s.Require().Len(spends, 1)
		s.Equal("as1", spends[0].Hash)
		s.Equal("DAG58AYZbPHyiYQcmD96T3zWK2HyN3Zc3cjWGqi7", spends[0].Destination)
		s.Equal(tx.Amount(500), spends[0].Amount)
		s.Equal(tx.Amount(10), spends[0].Fee)
		s.Equal(uint64(1000), spends[0].LastValidEpochProgress)
	})

	s.Run("sends active=true when ActiveOnly is set", func() {
		var capturedQuery string
		rt := transportFunc(func(r *http.Request) (*http.Response, error) {
			capturedQuery = r.URL.RawQuery
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"data":[],"meta":{"next":""}}`))}, nil
		})
		be := &blockExplorer{httpClient: &http.Client{Transport: rt}, beURL: "http://test-be"}

		_, _, _ = be.AllowSpendsByAddress(context.Background(), addr, PageOpts{ActiveOnly: true})
		s.Contains(capturedQuery, "active=true")
	})

	s.Run("returns ErrInvalidAddress on empty address", func() {
		be := &blockExplorer{httpClient: http.DefaultClient, beURL: "http://test-be"}
		_, _, err := be.AllowSpendsByAddress(context.Background(), "", PageOpts{})
		s.ErrorIs(err, ErrInvalidAddress)
	})
}
