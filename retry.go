// yaegi:tags purego
package retryif

import (
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"time"
)

const typeName = "RetryIf"

// retry is a middleware that retries requests.
type retry struct {
	config        	*RetryConfig
	next            http.Handler
	name            string
}

type RetryConfig struct {
	// Attempts defines how many times the request should be retried.
	Attempts int `json:"attempts,omitempty" toml:"attempts,omitempty" yaml:"attempts,omitempty" export:"true"`

	// StatusCodes defines the status codes which will trigger a retry if received
	StatusCodes []int `json:"statusCodes,omitempty" toml:"statusCodes,omitempty" yaml:"statusCodes,omitempty" export:"true"`
}

func CreateConfig() *RetryConfig {
	return &RetryConfig{
		Attempts: 2,
		StatusCodes: []int{503},
	}
}

// New returns a new retry middleware.
func New(ctx context.Context, next http.Handler, config *RetryConfig, name string) (http.Handler, error) {
	if config.Attempts <= 0 {
		return nil, fmt.Errorf("incorrect (or empty) value for attempt (%d)", config.Attempts)
	}

	return &retry{
		config: config,
		next:   next,
		name:   name,
	}, nil
}

func (r *retry) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if r.config.Attempts == 1 {
		r.next.ServeHTTP(rw, req)
		return
	}

	closableBody := req.Body
	if closableBody != nil {
		defer closableBody.Close()
	}

	// if we might make multiple attempts, swap the body for an io.NopCloser
	// cf https://github.com/traefik/traefik/issues/1008
	req.Body = io.NopCloser(closableBody)

	attempts := 1

	operation := func() error {
		shouldRetry := attempts < r.config.Attempts
		retryResponseWriter := newResponseWriter(rw, shouldRetry, r.config.StatusCodes, attempts-1)

		r.next.ServeHTTP(retryResponseWriter, req)

		if !retryResponseWriter.ShouldRetry() {
			return nil
		}

		attempts++
		return fmt.Errorf("attempt %d failed", attempts-1)
	}

	var err error
	for true {
		err = operation()
		if err == nil {
			break
		}
	}
}


