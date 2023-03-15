package retryif_test

import (
	"context"
	"github.com/fusionmedialimited/retryif"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRetryIfNoRetry(t *testing.T) {
	cfg := retryif.CreateConfig()
	cfg.StatusCodes = append(cfg.StatusCodes, http.StatusServiceUnavailable)

	ctx := context.Background()

	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(200)
	})

	handler, err := retryif.New(ctx, next, cfg, "retryif")
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost", nil)
	if err != nil {
		t.Fatal(err)
	}

	handler.ServeHTTP(recorder, req)
}

func TestRetryIfMaxAttempts(t *testing.T) {
	cfg := retryif.CreateConfig()
	cfg.Attempts = 3
	cfg.StatusCodes = append(cfg.StatusCodes, http.StatusServiceUnavailable)

	ctx := context.Background()

	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusServiceUnavailable)
	})

	handler, err := retryif.New(ctx, next, cfg, "retryif")
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost", nil)
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	retries := recorder.Header().Get("X-RetryIf-Retries")
	if retries != "2" {
		t.Errorf("Expected X-RetryIf-Retries header value to be '2', got '%s'", retries)
	}

	if recorder.Result().StatusCode != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d", recorder.Result().StatusCode)
	}
}

func TestRetryOnce(t *testing.T) {
	cfg := retryif.CreateConfig()
	cfg.Attempts = 3
	cfg.StatusCodes = append(cfg.StatusCodes, http.StatusServiceUnavailable)

	ctx := context.Background()

	attempt := 0
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if attempt == 0 {
			rw.WriteHeader(http.StatusServiceUnavailable)
			attempt = 1
		} else {
			rw.WriteHeader(http.StatusOK)
		}
	})

	handler, err := retryif.New(ctx, next, cfg, "retryif")
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost", nil)
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	retries := recorder.Header().Get("X-RetryIf-Retries")
	if retries != "1" {
		t.Errorf("Expected X-RetryIf-Retries header value to be '1', got '%s'", retries)
	}

	if recorder.Result().StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", recorder.Result().StatusCode)
	}
}

func TestRetryCarryHeaders(t *testing.T) {
	cfg := retryif.CreateConfig()
	cfg.Attempts = 3
	cfg.StatusCodes = append(cfg.StatusCodes, http.StatusServiceUnavailable)

	ctx := context.Background()

	attempt := 0
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if attempt == 0 {
			rw.WriteHeader(http.StatusServiceUnavailable)
			attempt = 1
		} else {
			rw.Header().Set("Location", "http://redirect.com")
			rw.WriteHeader(http.StatusMovedPermanently)
		}
	})

	handler, err := retryif.New(ctx, next, cfg, "retryif")
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost", nil)
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	retries := recorder.Header().Get("Location")
	if retries != "http://redirect.com" {
		t.Errorf("Expected Location header value to be carried", retries)
	}

	if recorder.Result().StatusCode != http.StatusMovedPermanently {
		t.Errorf("Expected status 301, got %d", recorder.Result().StatusCode)
	}
}

func TestNoRetryCarryHeaders(t *testing.T) {
	cfg := retryif.CreateConfig()
	cfg.StatusCodes = append(cfg.StatusCodes, http.StatusServiceUnavailable)

	ctx := context.Background()

	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Set("Location", "http://redirect.com")
		rw.WriteHeader(http.StatusMovedPermanently)
	})

	handler, err := retryif.New(ctx, next, cfg, "retryif")
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost", nil)
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	retries := recorder.Header().Get("Location")
	if retries != "http://redirect.com" {
		t.Errorf("Expected Location header value to be carried", retries)
	}

	if recorder.Result().StatusCode != http.StatusMovedPermanently {
		t.Errorf("Expected status 301, got %d", recorder.Result().StatusCode)
	}
}
