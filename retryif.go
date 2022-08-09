package retryif

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
)

type Config struct {
	Attempts        int   `json:"attempts"`
	Status          []int `json:"status"`
	InitialInterval int   `json:"initial_interval"`
}

func CreateConfig() *Config {
	return &Config{}
}

type RetryIF struct {
	name     string
	attempts int
	next     http.Handler
	Status   []int
}

func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {

	if len(config.Status) == 0 {
		return nil, fmt.Errorf("status is empty, please define at lease on Status code")
	}

	return &RetryIF{
		name:     name,
		next:     next,
		attempts: config.Attempts,
		Status:   config.Status,
	}, nil
}

func (r *RetryIF) ServeHTTP(rw http.ResponseWriter, req *http.Request) {

	if r.attempts == 1 {
		r.next.ServeHTTP(rw, req)
		return
	}

	attempts := 1
	code, body := r.testRequest(req)

	fmt.Printf("Got new Status: %d\n", code)

	if r.containsCode(code) {
		for attempts < r.attempts {
			attempts++

			attemptCode, attemptBody := r.testRequest(req)
			fmt.Printf("Got new Status: %d\n", attemptCode)

			if !r.containsCode(attemptCode) {
				rw.Write(attemptBody.Bytes())
				fmt.Printf("Request Got vaild staus code, new status code: %d, Attempts number: %d\n", attemptCode, attempts)
				break
			} else if attempts >= r.attempts && r.containsCode(attemptCode) {

				rw.WriteHeader(attemptCode)
				rw.Write(attemptBody.Bytes())

				fmt.Errorf("Could not get other Status, the Status is %d, Attempts number: %d\n", attemptCode, attempts)
				break
			}
		}
	} else {
		rw.Write(body.Bytes())
		fmt.Println("Successful in first attempt")
	}
}

func (r *RetryIF) containsCode(stCode int) bool {
	var exists bool = false
	for _, code := range r.Status {
		if code == stCode {
			exists = true
		}
	}
	return exists
}

func (r *RetryIF) testRequest(req *http.Request) (int, *bytes.Buffer) {
	recorder := httptest.NewRecorder()
	r.next.ServeHTTP(recorder, req)

	return recorder.Code, recorder.Body
}
