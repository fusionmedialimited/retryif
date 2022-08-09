package retryif

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
)

const (
	defaultErrMSG = "Service Unavailable"
	defaultstatus = 503
)

type Config struct {
	attempts     int    `json:"attempts"`
	Status       []int  `json:"status"`
	timeout      int    `json:"timeout,omitempty"`
	errorMessage string `json:"errorMessage,omitempty"`
}

func CreateConfig() *Config {
	return &Config{}
}

type RetryIF struct {
	name         string
	attempts     int
	next         http.Handler
	status       []int
	timeout      int
	errorMessage string
}

func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {

	fmt.Println(config)

	if len(config.Status) == 0 {
		return nil, fmt.Errorf("status is empty, please define at lease on status code")
	}

	return &RetryIF{
		name:         name,
		next:         next,
		attempts:     config.attempts,
		status:       config.Status,
		timeout:      config.timeout,
		errorMessage: config.errorMessage,
	}, nil
}

func (r *RetryIF) ServeHTTP(rw http.ResponseWriter, req *http.Request) {

	if r.attempts == 1 {
		r.next.ServeHTTP(rw, req)
		return
	}

	attempts := 1
	fmt.Println("Hello From Plugin")

	code, body := r.testRequest(req)

	fmt.Printf("Got new status: %d\n", code)

	if r.containsCode(code) {
		for attempts < r.attempts {
			attempts++

			attemptCode, attemptBody := r.testRequest(req)
			fmt.Printf("Got new status: %d", attemptCode)

			if r.containsCode(attemptCode) {
				rw.Write(attemptBody.Bytes())
				fmt.Printf("Got new status: %b", attemptBody)
				break
			} else if attempts >= r.attempts && r.containsCode(attemptCode) {
				fmt.Println("Could not get other status, the status is ", attemptCode)
				break
			}
		}
	} else {
		rw.Write(body.Bytes())
		fmt.Println("Successful in first attempt")
	}
}

func (r *RetryIF) containsCode(stCode int) bool {

	fmt.Println("RetryIf status list: ", r.status, " got staus from test: ", stCode)
	var exists bool = false
	for _, code := range r.status {
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
