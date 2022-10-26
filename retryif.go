package retryif

import (
	"compress/gzip"
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
	LoggerWarn  = log.New(io.Discard, "WARN: RetryIF: ", log.Ldate|log.Ltime|log.Lshortfile)
	LoggerError = log.New(io.Discard, "Error: RetryIF: ", log.Ldate|log.Ltime|log.Lshortfile)
)

type Config struct {
	Attempts int                 `json:"attempts"`
	Status   []int               `json:"status"`
	Headers  map[string][]string `json:"headers"`
	LogLevel string              `json:"loglevel"`
}

func CreateConfig() *Config {
	return &Config{
		Attempts: 2,
		Status:   []int{503},
		Headers:  make(map[string][]string),
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
		return nil, fmt.Errorf("status is empty, please define at least one of status")
	}

	// Set Default log level to info in case log level to defined
	switch config.LogLevel {
	case "ERROR":
		LoggerError.SetOutput(os.Stdout)
	case "WARN":
		LoggerError.SetOutput(os.Stdout)
		LoggerWarn.SetOutput(os.Stdout)
	case "INFO":
		LoggerError.SetOutput(os.Stdout)
		LoggerWarn.SetOutput(os.Stdout)
		LoggerInfo.SetOutput(os.Stdout)
	case "DEBUG":
		LoggerError.SetOutput(os.Stdout)
		LoggerWarn.SetOutput(os.Stdout)
		LoggerInfo.SetOutput(os.Stdout)
		LoggerDebug.SetOutput(os.Stdout)
	default:
		LoggerError.SetOutput(os.Stdout)
		LoggerWarn.SetOutput(os.Stdout)
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

	LoggerInfo.Printf("Got new Status: %d", respond.StatusCode)

	if r.containsCode(respond.StatusCode) {
		for attempts < r.attempts {
			attempts++

			attemptRespond := r.testRequest(req)
			LoggerInfo.Printf("Got new Status: %d\n", attemptRespond.StatusCode)

			if !r.containsCode(attemptRespond.StatusCode) {
				rw.WriteHeader(attemptRespond.StatusCode)

				attemptBody := getHttpBody(attemptRespond)

				rw.Write(attemptBody)

				LoggerInfo.Printf("Request Got valid status code, new status code: %d, Attempts number: %d\n", attemptRespond.StatusCode, attempts)

				PrintDebugResponse(req, attemptRespond)
				break
			} else if attempts >= r.attempts && r.containsCode(attemptRespond.StatusCode) {

				rw.WriteHeader(attemptRespond.StatusCode)

				attemptsBody := getHttpBody(attemptRespond)

				rw.Write(attemptsBody)

				LoggerInfo.Printf("Could not get other Status, the Status is %d, Attempts number: %d\n", attemptRespond.StatusCode, attempts)

				PrintDebugResponse(req, attemptRespond)
				break
			}
		}
	} else {
		rw.WriteHeader(respond.StatusCode)

		body := getHttpBody(respond)

		rw.Write(body)
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

func (r *RetryIF) testRequest(req *http.Request) *http.Response {
	recorder := httptest.NewRecorder()
	r.next.ServeHTTP(recorder, req)

	resp := recorder.Result()

	return resp
}

func getHttpBody(resp *http.Response) []byte {
	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		reader, err := gzip.NewReader(resp.Body)
		reader.Close()

		if err != nil {
			LoggerError.Println(err)
		}

		body, err := io.ReadAll(reader)

		if err != nil {
			LoggerError.Println(err)
		}

		return body

	default:
		body, err := io.ReadAll(resp.Body)

		if err != nil {
			LoggerError.Println(err)
		}

		return body
	}
}

func PrintDebugResponse(req *http.Request, res *http.Response) {
	// Print Request Headers:
	jsonReqHeaders, _ := json.Marshal(req.Header)
	LoggerDebug.Println("Request Headers: ", string(jsonReqHeaders))

	// Print Respond Headers:
	jsonRespondHeaders, _ := json.Marshal(res.Header)
	LoggerDebug.Println("Respond Headers: ", string(jsonRespondHeaders))

	// Print Respond Body:
	body, err := io.ReadAll(res.Body)
	if err != nil {
		LoggerError.Println(err)
	}
	LoggerDebug.Println(string(body))
}
