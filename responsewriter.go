package retryif

import (
	"fmt"
	"net"
	"bufio"
	"strconv"
	"net/http"
)

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
