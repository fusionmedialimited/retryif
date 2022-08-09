package retryif_test

import (
	"context"
	"github.com/fusionmedialimited/retryif"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRetryIf(t *testing.T) {
	cfg := retryif.CreateConfig()
	cfg.Status = append(cfg.Status, 503)

	ctx := context.Background()

	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {})

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
