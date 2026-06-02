package network

import (
	"net/http"
	"time"
)

type config struct {
	httpClient *http.Client
	l0URL      string
	l1URL      string
	beURL      string
}

// Option configures a Client or BlockExplorer at construction time.
type Option func(*config)

func newConfig(options ...Option) *config {
	cfg := &config{}
	for _, opt := range options {
		opt(cfg)
	}
	if cfg.httpClient == nil {
		cfg.httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return cfg
}

// WithHTTPClient sets the HTTP client to use for requests.
func WithHTTPClient(c *http.Client) Option {
	return func(cfg *config) {
		cfg.httpClient = c
	}
}

// WithMainNet configures the network endpoints to Constellation's production
// mainnet. Tokens transferred here are real value, so use with care.
func WithMainNet() Option {
	return func(cfg *config) {
		cfg.l0URL = "https://l0-lb-mainnet.constellationnetwork.io"
		cfg.l1URL = "https://l1-lb-mainnet.constellationnetwork.io"
		cfg.beURL = "https://be-mainnet.constellationnetwork.io"
	}
}

// WithIntegrationNet configures the network endpoints to Constellation's
// IntegrationNet, the pre-production environment for testing. Tokens here have
// no real value and can be obtained from the faucet at
// https://faucet.constellationnetwork.io/integrationnet/faucet/<DAG_ADDRESS>.
func WithIntegrationNet() Option {
	return func(cfg *config) {
		cfg.l0URL = "https://l0-lb-integrationnet.constellationnetwork.io"
		cfg.l1URL = "https://l1-lb-integrationnet.constellationnetwork.io"
		cfg.beURL = "https://be-integrationnet.constellationnetwork.io"
	}
}
