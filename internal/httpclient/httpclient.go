/*
Package httpclient implements http client.
*/
package httpclient

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"time"
)

var _ Doer = (*Client)(nil) // compile time proof

// defaults.
const (
	DefaultMaxIdleConns    = 10
	DefaultIdleConnTimeout = 10 * time.Second
	DefaultTimeout         = 60 * time.Second

	MaxIdleConnsMax    = 100
	IdleConnTimeoutMax = 60 * time.Second
	TimeoutMax         = 30 * time.Second

	UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36"
)

// sentinel errors.
var (
	ErrInvalid = errors.New("invalid value")
)

// Doer satisfies RoundTripper interface.
type Doer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Client holds http client params.
type Client struct {
	HTTPClient      *http.Client
	MaxIdleConns    int
	IdleConnTimeout time.Duration
	Timeout         time.Duration
}

// Do executes the given HTTP request using the underlying HTTP client.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	return c.HTTPClient.Do(req)
}

func (c *Client) setDefaults() {
	if c.MaxIdleConns <= 0 || c.MaxIdleConns > MaxIdleConnsMax {
		c.MaxIdleConns = DefaultMaxIdleConns
	}
	if c.IdleConnTimeout <= 0 || c.IdleConnTimeout > IdleConnTimeoutMax {
		c.IdleConnTimeout = DefaultIdleConnTimeout
	}
	if c.Timeout <= 0 || c.Timeout > TimeoutMax {
		c.Timeout = DefaultTimeout
	}
}

// Option represents option function type.
type Option func(*Client) error

// WithMaxIdleConns sets MaxIdleConns.
func WithMaxIdleConns(n int) Option {
	return func(c *Client) error {
		if n <= 0 || n > MaxIdleConnsMax {
			return fmt.Errorf("%w, '%d' is not valid", ErrInvalid, n)
		}

		c.MaxIdleConns = n

		return nil
	}
}

// WithIdleConnTimeout sets IdleConnTimeout.
func WithIdleConnTimeout(d time.Duration) Option {
	return func(c *Client) error {
		if d < 0 {
			return fmt.Errorf("%w, '%d' must > 0", ErrInvalid, d)
		}

		if d > IdleConnTimeoutMax {
			return fmt.Errorf("%w, '%d' must < %d", ErrInvalid, d, IdleConnTimeoutMax)
		}

		c.IdleConnTimeout = d

		return nil
	}
}

// WithTimeout sets the client timeout with a max limit.
func WithTimeout(d time.Duration) Option {
	return func(c *Client) error {
		if d <= 0 || d > TimeoutMax {
			return fmt.Errorf("%w: timeout must be between 1s and %s, got %s", ErrInvalid, TimeoutMax, d)
		}
		c.Timeout = d
		return nil
	}
}

// New instantiates new http client instance.
func New(options ...Option) (*Client, error) {
	client := new(Client)
	client.setDefaults()

	for _, option := range options {
		if err := option(client); err != nil {
			return nil, err
		}
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxIdleConns = client.MaxIdleConns
	transport.IdleConnTimeout = client.IdleConnTimeout
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint

	cl := &http.Client{
		Transport: transport,
		Timeout:   client.Timeout,
	}
	client.HTTPClient = cl

	return client, nil
}
