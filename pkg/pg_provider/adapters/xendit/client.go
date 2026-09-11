package xendit

import (
	"context"
	"encoding/base64"
	"net/http"
	"time"

	"github.com/cenkalti/backoff/v5"
)

const (
	defaultTimeout     = 30 * time.Second
	defaultRetryMax    = 2
	defaultBaseDelay   = 500 * time.Millisecond
	defaultMaxDelay    = 5 * time.Second
	requestPath        = "identity/v2/bank_account_validation"
	authorizationBasic = "Basic "
)

type poster interface {
	PostWithHeaders(ctx context.Context, path string, queryParams map[string]any, body []byte,
		headers map[string]string) (*http.Response, error)
}

type Client struct {
	config Config
	client poster
}

func NewClient(cfg Config, c poster) *Client {
	if cfg.RetryMax == 0 {
		cfg.RetryMax = defaultRetryMax
	}
	if cfg.RetryBaseDelay == 0 {
		cfg.RetryBaseDelay = defaultBaseDelay
	}
	if cfg.RetryMaxDelay == 0 {
		cfg.RetryMaxDelay = defaultMaxDelay
	}

	return &Client{
		config: cfg,
		client: c,
	}
}

func (c *Client) basicAuth() string {
	raw := c.config.APIKey + ":" + c.config.APISecret
	return authorizationBasic + base64.StdEncoding.EncodeToString([]byte(raw))
}

func (c *Client) backoff() *backoff.ExponentialBackOff {
	bo := backoff.NewExponentialBackOff()
	bo.InitialInterval = c.config.RetryBaseDelay
	bo.MaxInterval = c.config.RetryMaxDelay

	return bo
}
