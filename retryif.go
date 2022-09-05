package retryif

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
)

var (
	LoggerInfo  = log.New(io.Discard, "INFO: RetryIF: ", log.Ldate|log.Ltime|log.Lshortfile)
	LoggerDebug = log.New(io.Discard, "DEBUG: RetryIF: ", log.Ldate|log.Ltime|log.Lshortfile)
	LoggerError = log.New(io.Discard, "Error: RetryIF: ", log.Ldate|log.Ltime|log.Lshortfile)
)

type Config struct {
	Attempts int    `json:"attempts"`
	Status   []int  `json:"status"`
	LogLevel string `json:"loglevel"`
}

func CreateConfig() *Config {
	return &Config{
		Attempts: 2,
		Status:   []int{503},
		LogLevel: "INFO",
	}
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
	respond := r.testRequest(req)

	LoggerInfo.Printf("Got new Status: %d", respond.Code)

	if r.containsCode(respond.Code) {
		for attempts < r.attempts {
			attempts++

			attemptRespond := r.testRequest(req)
			LoggerInfo.Printf("Got new Status: %d\n", attemptRespond.Code)

			if !r.containsCode(attemptRespond.Code) {
				rw.WriteHeader(attemptRespond.Code)
				rw.Write(attemptRespond.Body.Bytes())

				LoggerInfo.Printf("Request Got valid status code, new status code: %d, Attempts number: %d\n", attemptRespond.Code, attempts)

				PrintDebugResponse(req, attemptRespond)
				break
			} else if attempts >= r.attempts && r.containsCode(attemptRespond.Code) {

				rw.WriteHeader(attemptRespond.Code)
				rw.Write(attemptRespond.Body.Bytes())

				LoggerInfo.Printf("Could not get other Status, the Status is %d, Attempts number: %d\n", attemptRespond.Code, attempts)

				PrintDebugResponse(req, attemptRespond)
				break
			}
		}
	} else {
		rw.WriteHeader(respond.Code)
		rw.Write(respond.Body.Bytes())
		LoggerInfo.Print("Request passed successfully in first attempt :)")

		PrintDebugResponse(req, respond)
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

func (r *RetryIF) testRequest(req *http.Request) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	r.next.ServeHTTP(recorder, req)

	return recorder
}

func PrintDebugResponse(req *http.Request, res *httptest.ResponseRecorder) {
	// Print Request Headers:
	jsonReqHeaders, _ := json.Marshal(req.Header)
	LoggerDebug.Println("Request Headers: ", string(jsonReqHeaders))

	// Print Respond Headers:
	jsonRespondHeaders, _ := json.Marshal(res.Result().Header)
	LoggerDebug.Println("Respond Headers: ", string(jsonRespondHeaders))

	// Print Respond Body:
	LoggerDebug.Println(string(res.Body.Bytes()))
}
