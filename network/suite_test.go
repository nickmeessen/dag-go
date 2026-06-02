package network

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
)

func TestSuite(t *testing.T) {
	suite.Run(t, new(ClientTestSuite))
	suite.Run(t, new(BlockExplorerTestSuite))
}

// transportFunc adapts a function to the http.RoundTripper interface so tests
// can capture outgoing requests without needing a full httptest server.
type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

// captureURL returns an http.Client that records the requested URL into got
// and responds with an empty data/meta envelope. Use for asserting routing.
func captureURL(got *string) *http.Client {
	rt := transportFunc(func(r *http.Request) (*http.Response, error) {
		*got = r.URL.String()
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(`{"data":{},"meta":{"next":""}}`)),
		}, nil
	})
	return &http.Client{Transport: rt}
}
