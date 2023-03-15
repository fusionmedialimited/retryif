package retryif

import (
	"context"
	"fmt"
	"io"
	"math"
	"net"
	"bufio"
	"strconv"
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


func newResponseWriter(rw http.ResponseWriter, shouldRetry bool, codes []int, attempt int) *responseWriter {
	return &responseWriter{
		responseWriter: rw,
		headers:        make(http.Header),
		shouldRetry:    shouldRetry,
		statusCodes:    codes,
		attempt:        attempt,
	}
}

type responseWriter struct {
	attempt        int
	statusCodes    []int
	responseWriter http.ResponseWriter
	headers        http.Header
	shouldRetry    bool
	written        bool
}

func (r *responseWriter) ShouldRetry() bool {
	return r.shouldRetry
}

func (r *responseWriter) DisableRetries() {
	r.shouldRetry = false
}

func (r *responseWriter) Header() http.Header {
	if r.written {
		return r.responseWriter.Header()
	}
	return r.headers
}

func (r *responseWriter) Write(buf []byte) (int, error) {
	if r.ShouldRetry() {
		return len(buf), nil
	}
	return r.responseWriter.Write(buf)
}

func (r *responseWriter) WriteHeader(code int) {
	if r.ShouldRetry() && !r.ShouldRetryOnCode(code) {
		r.DisableRetries()
	}

	if r.ShouldRetry() {
		return
	}

	// In that case retry case is set to false which means we at least managed
	// to write headers to the backend : we are not going to perform any further retry.
	// So it is now safe to alter current response headers with headers collected during
	// the latest try before writing headers to client.
	headers := r.responseWriter.Header()
	headers.Set("X-RetryIf-Retries", strconv.Itoa(r.attempt))
	for header, value := range r.headers {
		headers[header] = value
	}

	r.responseWriter.WriteHeader(code)
	r.written = true
}

func (r *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := r.responseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("%T is not a http.Hijacker", r.responseWriter)
	}
	return hijacker.Hijack()
}

func (r *responseWriter) Flush() {
	if flusher, ok := r.responseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (r *responseWriter) ShouldRetryOnCode(stCode int) bool {
	for _, code := range r.statusCodes {
		if code == stCode {
			return true
		}
	}
	return false
}

func (r *responseWriter) CloseNotify() <-chan bool {
	return r.responseWriter.(http.CloseNotifier).CloseNotify()
}
