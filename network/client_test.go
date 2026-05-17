package network

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-errors/errors"
	"github.com/nickmeessen/dag-go/tx"
	"github.com/nickmeessen/dag-go/wallet"
	"github.com/stretchr/testify/suite"
)

type ClientTestSuite struct {
	suite.Suite
}

func TestClientTestSuite(t *testing.T) {
	suite.Run(t, new(ClientTestSuite))
}

func (s *ClientTestSuite) TestNew() {
	s.Run("routes balance call to mainnet L0", func() {
		w, err := wallet.New()
		s.Require().NoError(err)

		var capturedURL string
		rt := transportFunc(func(r *http.Request) (*http.Response, error) {
			capturedURL = r.URL.String()
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(`{"balance":0,"ordinal":0}`)),
			}, nil
		})
		c, err := New(WithMainNet(), WithHTTPClient(&http.Client{Transport: rt}))
		s.Require().NoError(err)

		_, _ = c.Balance(context.Background(), w.Address())
		s.Contains(capturedURL, "https://l0-lb-mainnet.constellationnetwork.io/dag/"+w.Address()+"/balance")
	})

	s.Run("routes balance call to integrationNet L0", func() {
		w, err := wallet.New()
		s.Require().NoError(err)

		var capturedURL string
		rt := transportFunc(func(r *http.Request) (*http.Response, error) {
			capturedURL = r.URL.String()
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(`{"balance":0,"ordinal":0}`)),
			}, nil
		})
		c, err := New(WithIntegrationNet(), WithHTTPClient(&http.Client{Transport: rt}))
		s.Require().NoError(err)

		_, _ = c.Balance(context.Background(), w.Address())
		s.Contains(capturedURL, "https://l0-lb-integrationnet.constellationnetwork.io/dag/"+w.Address()+"/balance")
	})

	s.Run("fails with missing network config", func() {
		_, err := New()
		s.Require().Error(err)
		s.ErrorIs(err, ErrMissingNetworkConfiguration)
	})

	s.Run("uses custom HTTP client when provided", func() {
		w, err := wallet.New()
		s.Require().NoError(err)

		called := false
		rt := transportFunc(func(_ *http.Request) (*http.Response, error) {
			called = true
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(`{"balance":0,"ordinal":0}`)),
			}, nil
		})
		c, err := New(WithIntegrationNet(), WithHTTPClient(&http.Client{Transport: rt}))
		s.Require().NoError(err)

		_, _ = c.Balance(context.Background(), w.Address())
		s.True(called)
	})
}

func (s *ClientTestSuite) TestNewMetagraphClient() {
	s.Run("routes balance call to metagraph L0 with currency path", func() {
		w, err := wallet.New()
		s.Require().NoError(err)

		var capturedURL string
		rt := transportFunc(func(r *http.Request) (*http.Response, error) {
			capturedURL = r.URL.String()
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(`{"balance":0,"ordinal":0}`)),
			}, nil
		})
		c, err := NewMetagraphClient(
			"http://test-l0",
			"http://test-l1",
			WithHTTPClient(&http.Client{Transport: rt}),
		)
		s.Require().NoError(err)

		_, _ = c.Balance(context.Background(), w.Address())
		s.Contains(capturedURL, "http://test-l0/currency/"+w.Address()+"/balance")
	})

	s.Run("fails when l0URL is empty", func() {
		_, err := NewMetagraphClient("", "http://test-l1")
		s.ErrorIs(err, ErrMissingNetworkConfiguration)
	})

	s.Run("fails when l1URL is empty", func() {
		_, err := NewMetagraphClient("http://test-l0", "")
		s.ErrorIs(err, ErrMissingNetworkConfiguration)
	})

	s.Run("uses custom HTTP client when provided", func() {
		w, err := wallet.New()
		s.Require().NoError(err)

		called := false
		rt := transportFunc(func(_ *http.Request) (*http.Response, error) {
			called = true
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(`{"balance":0,"ordinal":0}`)),
			}, nil
		})
		c, err := NewMetagraphClient(
			"http://test-l0",
			"http://test-l1",
			WithHTTPClient(&http.Client{Transport: rt}),
		)
		s.Require().NoError(err)

		_, _ = c.Balance(context.Background(), w.Address())
		s.True(called)
	})
}

func (s *ClientTestSuite) TestDoJSON() {
	s.Run("decodes JSON on 200", func() {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"result":"ok"}`))
		}))
		defer srv.Close()

		c := &client{httpClient: srv.Client(), l0URL: srv.URL, l1URL: srv.URL, balancePath: "/dag/"}

		var out struct {
			Result string `json:"result"`
		}
		err := c.doJSON(context.Background(), "send", srv.URL, http.MethodPost, nil, &out)

		s.Require().NoError(err)
		s.Equal("ok", out.Result)
	})

	s.Run("errors on transport failure", func() {
		c := &client{httpClient: &http.Client{Timeout: 1 * time.Millisecond}, l0URL: "http://invalid-url", l1URL: "http://invalid-url"}

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

		c := &client{httpClient: srv.Client(), l0URL: srv.URL, l1URL: srv.URL, balancePath: "/dag/"}

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

		c := &client{httpClient: srv.Client(), l0URL: srv.URL, l1URL: srv.URL, balancePath: "/dag/"}

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

		c := &client{httpClient: srv.Client(), l0URL: srv.URL, l1URL: srv.URL, balancePath: "/dag/"}

		var out struct{}
		err := c.doJSON(context.Background(), "send", srv.URL, http.MethodPost, nil, &out)

		s.Require().Error(err)
		s.True(errors.Is(err, ErrTxRejected))
		s.Contains(err.Error(), "send")
		s.Contains(err.Error(), "418")
	})

	s.Run("errors when body can't be marshaled", func() {
		c := &client{httpClient: http.DefaultClient, l0URL: "http://test-url", l1URL: ""}

		var out struct{}
		err := c.doJSON(context.Background(), "send", "http://x", http.MethodPost, make(chan int), &out)

		s.Require().Error(err)
		s.Contains(err.Error(), "marshal request body")
	})

	s.Run("errors on invalid HTTP method", func() {
		c := &client{httpClient: http.DefaultClient, l0URL: "http://test-url", l1URL: ""}

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

		c := &client{httpClient: srv.Client(), l0URL: srv.URL, l1URL: srv.URL, balancePath: "/dag/"}

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
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"balance": 12345, "ordinal": 5}`))
		}))
		defer srv.Close()

		c, err := NewMetagraphClient(srv.URL, srv.URL, WithHTTPClient(srv.Client()))
		s.Require().NoError(err)

		bal, err := c.Balance(context.Background(), "DAG3jifKUZPc213rRLSfZVSLZfPRfX7fwTGh8tsy")
		s.Require().NoError(err)
		s.Equal(tx.Datoshi(12345), bal)
	})

	s.Run("propagates ErrNotFound on 404", func() {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer srv.Close()

		c, err := NewMetagraphClient(srv.URL, srv.URL, WithHTTPClient(srv.Client()))
		s.Require().NoError(err)

		_, err = c.Balance(context.Background(), "DAG3jifKUZPc213rRLSfZVSLZfPRfX7fwTGh8tsy")
		s.ErrorIs(err, ErrNotFound)
	})

	s.Run("rejects empty address without making HTTP call", func() {
		c, err := NewMetagraphClient("http://test-url", "http://test-url")
		s.Require().NoError(err)

		_, err = c.Balance(context.Background(), "")
		s.ErrorIs(err, ErrInvalidAddress)
	})

	s.Run("hits /currency/{addr}/balance in metagraph mode", func() {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			s.Equal("/currency/DAG3jifKUZPc213rRLSfZVSLZfPRfX7fwTGh8tsy/balance", r.URL.Path)
			_, _ = w.Write([]byte(`{"balance": 999, "ordinal": 1}`))
		}))
		defer srv.Close()

		c, err := NewMetagraphClient(srv.URL, srv.URL, WithHTTPClient(srv.Client()))
		s.Require().NoError(err)

		bal, err := c.Balance(context.Background(), "DAG3jifKUZPc213rRLSfZVSLZfPRfX7fwTGh8tsy")
		s.Require().NoError(err)
		s.Equal(tx.Datoshi(999), bal)
	})
}

func (s *ClientTestSuite) TestLastTxRef() {
	s.Run("decodes ref from L1", func() {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			s.Equal("/transactions/last-reference/DAG3jifKUZPc213rRLSfZVSLZfPRfX7fwTGh8tsy", r.URL.Path)
			_, _ = w.Write([]byte(`{"hash":"abc","ordinal":42}`))
		}))
		defer srv.Close()

		c, err := NewMetagraphClient(srv.URL, srv.URL, WithHTTPClient(srv.Client()))
		s.Require().NoError(err)

		ref, err := c.LastTxRef(context.Background(), "DAG3jifKUZPc213rRLSfZVSLZfPRfX7fwTGh8tsy")
		s.Require().NoError(err)
		s.Equal(tx.Ref{Hash: "abc", Ordinal: 42}, ref)
	})

	s.Run("returns zero ref for first-time sender", func() {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"hash":"0000000000000000000000000000000000000000000000000000000000000000","ordinal":0}`))
		}))
		defer srv.Close()

		c, err := NewMetagraphClient(srv.URL, srv.URL, WithHTTPClient(srv.Client()))
		s.Require().NoError(err)

		ref, err := c.LastTxRef(context.Background(), "DAG3jifKUZPc213rRLSfZVSLZfPRfX7fwTGh8tsy")
		s.Require().NoError(err)
		s.Equal(uint64(0), ref.Ordinal)
		s.Equal("0000000000000000000000000000000000000000000000000000000000000000", ref.Hash)
	})

	s.Run("rejects empty address without making HTTP call", func() {
		c, err := NewMetagraphClient("http://test-url", "http://test-url")
		s.Require().NoError(err)

		_, err = c.LastTxRef(context.Background(), "")
		s.ErrorIs(err, ErrInvalidAddress)
	})

	s.Run("propagates ErrNotFound on 404", func() {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer srv.Close()

		c, err := NewMetagraphClient(srv.URL, srv.URL, WithHTTPClient(srv.Client()))
		s.Require().NoError(err)

		_, err = c.LastTxRef(context.Background(), "DAG3jifKUZPc213rRLSfZVSLZfPRfX7fwTGh8tsy")
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

		c, err := NewMetagraphClient(srv.URL, srv.URL, WithHTTPClient(srv.Client()))
		s.Require().NoError(err)

		hash, err := c.Send(context.Background(), signed)
		s.Require().NoError(err)
		s.Equal("tx-hash-123", hash)
	})

	s.Run("propagates ErrTxRejected on 4xx", func() {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
		}))
		defer srv.Close()

		c, err := NewMetagraphClient(srv.URL, srv.URL, WithHTTPClient(srv.Client()))
		s.Require().NoError(err)

		_, err = c.Send(context.Background(), signed)
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

		c, err := NewMetagraphClient(srv.URL, srv.URL, WithHTTPClient(srv.Client()))
		s.Require().NoError(err)

		ref, err := c.PendingTx(context.Background(), "abc123")
		s.Require().NoError(err)
		s.Equal(tx.Ref{Hash: "abc123", Ordinal: 7}, ref)
	})

	s.Run("returns ErrNotFound when tx left the pool (confirmation signal)", func() {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer srv.Close()

		c, err := NewMetagraphClient(srv.URL, srv.URL, WithHTTPClient(srv.Client()))
		s.Require().NoError(err)

		_, err = c.PendingTx(context.Background(), "abc123")
		s.ErrorIs(err, ErrNotFound)
	})
}

// transportFunc adapts a function to the http.RoundTripper interface so tests
// can capture outgoing requests without needing a full httptest server.
type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
