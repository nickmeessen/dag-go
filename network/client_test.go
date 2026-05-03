package network

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-errors/errors"
	"github.com/nickmeessen/dag-go/tx"
	"github.com/stretchr/testify/suite"
)

type ClientTestSuite struct {
	suite.Suite
}

func TestClientTestSuite(t *testing.T) {
	suite.Run(t, new(ClientTestSuite))
}

func (s *ClientTestSuite) TestNew() {
	s.Run("creates client for mainnet", func() {
		c, err := New(WithMainNet())
		s.Require().NoError(err)
		s.Equal("https://l0-lb-mainnet.constellationnetwork.io", c.l0URL)
		s.Equal("https://l1-lb-mainnet.constellationnetwork.io", c.l1URL)
	})

	s.Run("creates client for integrationNet", func() {
		c, err := New(WithIntegrationNet())
		s.Require().NoError(err)
		s.Equal("https://l0-lb-integrationnet.constellationnetwork.io", c.l0URL)
		s.Equal("https://l1-lb-integrationnet.constellationnetwork.io", c.l1URL)
	})

	s.Run("fails with missing network config", func() {
		_, err := New()
		s.Require().Error(err)
		s.ErrorIs(err, ErrMissingNetworkConfiguration)
	})

	s.Run("uses custom HTTP client when provided", func() {
		c, err := New(WithIntegrationNet(), WithHTTPClient(&http.Client{Timeout: 1 * time.Second}))
		s.Require().NoError(err)
		s.NotNil(c)
		s.Equal(1*time.Second, c.httpClient.Timeout)
	})
}

func (s *ClientTestSuite) TestDoJSON() {
	s.Run("decodes JSON on 200", func() {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"result":"ok"}`))
		}))
		defer srv.Close()

		c := &Client{httpClient: srv.Client(), l0URL: srv.URL, l1URL: srv.URL}

		var out struct {
			Result string `json:"result"`
		}
		err := c.doJSON(context.Background(), "send", srv.URL, http.MethodPost, nil, &out)

		s.Require().NoError(err)
		s.Equal("ok", out.Result)
	})

	s.Run("errors on transport failure", func() {
		c := &Client{httpClient: &http.Client{Timeout: 1 * time.Millisecond}, l0URL: "http://invalid-url", l1URL: "http://invalid-url"}

		var out struct{}
		err := c.doJSON(context.Background(), "send", "http://invalid-url", http.MethodPost, nil, &out)

		s.Require().Error(err)
		s.False(errors.Is(err, ErrTxRejected))
	})

	s.Run("errors on malformed JSON response", func() {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`invalid json`))
		}))
		defer srv.Close()

		c := &Client{httpClient: srv.Client(), l0URL: srv.URL, l1URL: srv.URL}

		var out struct{}
		err := c.doJSON(context.Background(), "send", srv.URL, http.MethodPost, nil, &out)

		s.Require().Error(err)
		s.False(errors.Is(err, ErrTxRejected))
	})

	s.Run("returns ErrTxRejected on 400", func() {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
		}))
		defer srv.Close()

		c := &Client{httpClient: srv.Client(), l0URL: srv.URL, l1URL: srv.URL}

		var out struct{}
		err := c.doJSON(context.Background(), "send", srv.URL, http.MethodPost, nil, &out)

		s.Require().Error(err)
		s.True(errors.Is(err, ErrTxRejected))
		s.Contains(err.Error(), "send")
		s.Contains(err.Error(), "400")
	})

	s.Run("returns ErrTxRejected on 418 (any 4xx)", func() {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusTeapot)
		}))
		defer srv.Close()

		c := &Client{httpClient: srv.Client(), l0URL: srv.URL, l1URL: srv.URL}

		var out struct{}
		err := c.doJSON(context.Background(), "send", srv.URL, http.MethodPost, nil, &out)

		s.Require().Error(err)
		s.True(errors.Is(err, ErrTxRejected))
		s.Contains(err.Error(), "send")
		s.Contains(err.Error(), "418")
	})

	s.Run("errors when body can't be marshaled", func() {
		c := &Client{httpClient: http.DefaultClient, l0URL: "http://test-url", l1URL: ""}

		var out struct{}
		err := c.doJSON(context.Background(), "send", "http://x", http.MethodPost, make(chan int), &out)

		s.Require().Error(err)
		s.Contains(err.Error(), "marshal request body")
	})

	s.Run("errors on invalid HTTP method", func() {
		c := &Client{httpClient: http.DefaultClient, l0URL: "http://test-url", l1URL: ""}

		var out struct{}
		err := c.doJSON(context.Background(), "send", "http://test-url", "BAD METHOD", nil, &out)

		s.Require().Error(err)
		s.Contains(err.Error(), "build request")
	})

	s.Run("returns generic error on 5xx", func() {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer srv.Close()

		c := &Client{httpClient: srv.Client(), l0URL: srv.URL, l1URL: srv.URL}

		var out struct{}
		err := c.doJSON(context.Background(), "send", srv.URL, http.MethodPost, nil, &out)

		s.Require().Error(err)
		s.False(errors.Is(err, ErrTxRejected))
		s.False(errors.Is(err, ErrNotFound))
		s.Contains(err.Error(), "500")
	})
}

func (s *ClientTestSuite) TestBalance() {
	s.Run("decodes balance from L0", func() {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			s.Equal("/dag/DAG3jifKUZPc213rRLSfZVSLZfPRfX7fwTGh8tsy/balance", r.URL.Path)
			_, _ = w.Write([]byte(`{"balance": 12345, "ordinal": 5}`))
		}))
		defer srv.Close()

		c := &Client{httpClient: srv.Client(), l0URL: srv.URL, l1URL: srv.URL}

		bal, err := c.Balance(context.Background(), "DAG3jifKUZPc213rRLSfZVSLZfPRfX7fwTGh8tsy")
		s.Require().NoError(err)
		s.Equal(tx.Datoshi(12345), bal)
	})

	s.Run("propagates ErrNotFound on 404", func() {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer srv.Close()

		c := &Client{httpClient: srv.Client(), l0URL: srv.URL, l1URL: srv.URL}

		_, err := c.Balance(context.Background(), "DAG3jifKUZPc213rRLSfZVSLZfPRfX7fwTGh8tsy")
		s.ErrorIs(err, ErrNotFound)
	})

	s.Run("rejects empty address without making HTTP call", func() {
		c := &Client{httpClient: http.DefaultClient, l0URL: "http://test-url", l1URL: ""}

		_, err := c.Balance(context.Background(), "")
		s.ErrorIs(err, ErrInvalidAddress)
	})
}

func (s *ClientTestSuite) TestLastTxRef() {
	s.Run("decodes ref from L1", func() {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			s.Equal("/transactions/last-reference/DAG3jifKUZPc213rRLSfZVSLZfPRfX7fwTGh8tsy", r.URL.Path)
			_, _ = w.Write([]byte(`{"hash":"abc","ordinal":42}`))
		}))
		defer srv.Close()

		c := &Client{httpClient: srv.Client(), l0URL: srv.URL, l1URL: srv.URL}

		ref, err := c.LastTxRef(context.Background(), "DAG3jifKUZPc213rRLSfZVSLZfPRfX7fwTGh8tsy")
		s.Require().NoError(err)
		s.Equal(tx.Ref{Hash: "abc", Ordinal: 42}, ref)
	})

	s.Run("returns zero ref for first-time sender", func() {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"hash":"0000000000000000000000000000000000000000000000000000000000000000","ordinal":0}`))
		}))
		defer srv.Close()

		c := &Client{httpClient: srv.Client(), l0URL: srv.URL, l1URL: srv.URL}

		ref, err := c.LastTxRef(context.Background(), "DAG3jifKUZPc213rRLSfZVSLZfPRfX7fwTGh8tsy")
		s.Require().NoError(err)
		s.Equal(uint64(0), ref.Ordinal)
		s.Equal("0000000000000000000000000000000000000000000000000000000000000000", ref.Hash)
	})

	s.Run("rejects empty address without making HTTP call", func() {
		c := &Client{httpClient: http.DefaultClient, l1URL: "http://test-url"}

		_, err := c.LastTxRef(context.Background(), "")
		s.ErrorIs(err, ErrInvalidAddress)
	})

	s.Run("propagates ErrNotFound on 404", func() {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer srv.Close()

		c := &Client{httpClient: srv.Client(), l0URL: srv.URL, l1URL: srv.URL}

		_, err := c.LastTxRef(context.Background(), "DAG3jifKUZPc213rRLSfZVSLZfPRfX7fwTGh8tsy")
		s.ErrorIs(err, ErrNotFound)
	})
}

func (s *ClientTestSuite) TestSend() {
	signed := tx.Signed{
		Value: tx.Transfer{
			Source:      "DAG3jifKUZPc213rRLSfZVSLZfPRfX7fwTGh8tsy",
			Destination: "DAG1ATvdAxGz4DNzPdrk6p8QD1CdSXNLvYypaaVQ",
			Amount:      tx.Datoshi(100000),
			Fee:         tx.Datoshi(10000),
			Parent:      tx.Ref{Hash: "0000000000000000000000000000000000000000000000000000000000000000", Ordinal: 0},
			Salt:        8725724278030335,
		},
		Proofs: []tx.Proof{{ID: "peer-id", Signature: "sig"}},
	}

	s.Run("posts signed tx to L1 and returns hash", func() {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			s.Equal(http.MethodPost, r.Method)
			s.Equal("/transactions", r.URL.Path)

			body, err := io.ReadAll(r.Body)
			s.Require().NoError(err)
			var got tx.Signed
			s.Require().NoError(json.Unmarshal(body, &got))
			s.Equal(signed, got)

			_, _ = w.Write([]byte(`{"hash":"tx-hash-123"}`))
		}))
		defer srv.Close()

		c := &Client{httpClient: srv.Client(), l0URL: srv.URL, l1URL: srv.URL}

		hash, err := c.Send(context.Background(), signed)
		s.Require().NoError(err)
		s.Equal("tx-hash-123", hash)
	})

	s.Run("propagates ErrTxRejected on 4xx", func() {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
		}))
		defer srv.Close()

		c := &Client{httpClient: srv.Client(), l0URL: srv.URL, l1URL: srv.URL}

		_, err := c.Send(context.Background(), signed)
		s.ErrorIs(err, ErrTxRejected)
	})
}

func (s *ClientTestSuite) TestPendingTx() {
	s.Run("decodes ref for pending tx", func() {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			s.Equal("/transactions/pending/abc123", r.URL.Path)
			_, _ = w.Write([]byte(`{"hash":"abc123","ordinal":7}`))
		}))
		defer srv.Close()

		c := &Client{httpClient: srv.Client(), l0URL: srv.URL, l1URL: srv.URL}

		ref, err := c.PendingTx(context.Background(), "abc123")
		s.Require().NoError(err)
		s.Equal(tx.Ref{Hash: "abc123", Ordinal: 7}, ref)
	})

	s.Run("returns ErrNotFound when tx left the pool (confirmation signal)", func() {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer srv.Close()

		c := &Client{httpClient: srv.Client(), l0URL: srv.URL, l1URL: srv.URL}

		_, err := c.PendingTx(context.Background(), "abc123")
		s.ErrorIs(err, ErrNotFound)
	})
}
