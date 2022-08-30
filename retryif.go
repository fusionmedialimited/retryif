package retryif

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
)

var (
	LoggerInfo  = log.New(io.Discard, "retryif [INFO]: ", log.Ldate|log.Ltime|log.Lshortfile)
	LoggerDebug = log.New(io.Discard, "retryif [DEBUG]: ", log.Ldate|log.Ltime|log.Lshortfile)
	LoggerError = log.New(io.Discard, "retryif [Error]: ", log.Ldate|log.Ltime|log.Lshortfile)
)

type Config struct {
	Attempts int    `json:"attempts"`
	Status   []int  `json:"status"`
	LogLevel string `json:"log_level,omitempty"`
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
		return nil, fmt.Errorf("status is empty, please define at least on Status code")
	}
	// Set Default log level to info in case log level to defined
	switch config.LogLevel {
	case "ERROR":
		LoggerError.SetOutput(os.Stdout)
	case "INFO":
		LoggerError.SetOutput(os.Stdout)
		LoggerInfo.SetOutput(os.Stdout)
	case "DEBUG":
		LoggerError.SetOutput(os.Stdout)
		LoggerInfo.SetOutput(os.Stdout)
		LoggerDebug.SetOutput(os.Stdout)
	default:
		LoggerError.SetOutput(os.Stdout)
		LoggerInfo.SetOutput(os.Stdout)
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

	LoggerInfo.Printf("Got new Status: %d", code)
	LoggerDebug.Printf("%s", string(body.Bytes()))

	if r.containsCode(code) {
		for attempts < r.attempts {
			attempts++

			attemptCode, attemptBody := r.testRequest(req)
			LoggerInfo.Printf("Got new Status: %d\n", attemptCode)

			if !r.containsCode(attemptCode) {
				rw.WriteHeader(attemptCode)
				rw.Write(attemptBody.Bytes())

				LoggerInfo.Printf("Request Got vaild staus code, new status code: %d, Attempts number: %d\n", attemptCode, attempts)
				break
			} else if attempts >= r.attempts && r.containsCode(attemptCode) {

				rw.WriteHeader(attemptCode)
				rw.Write(attemptBody.Bytes())

				LoggerInfo.Printf("Could not get other Status, the Status is %d, Attempts number: %d\n", attemptCode, attempts)
				break
			}
		}
	} else {
		rw.Write(body.Bytes())
		LoggerInfo.Print("Request passed successfully in first attempt :)")
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
