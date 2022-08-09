package retryif

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

const (
	defaultErrMSG = "Service Unavailable"
	defaultStatus = 503
)

var (
	LoggerDEBUG = log.New(ioutil.Discard, "DEBUG: retryif: ", log.Ldate|log.Ltime|log.Lshortfile)
	LoggerERROR = log.New(ioutil.Discard, "ERROR: retryif: ", log.Ldate|log.Ltime|log.Lshortfile)
	LoggerINFO  = log.New(ioutil.Discard, "INFO: retryif: ", log.Ldate|log.Ltime|log.Lshortfile)
)

type Config struct {
	attempts     int    `json:"attempts,omitempty",yaml:"attempts,omitempty"`
	Status       []int  `json:"statusCode,omitempty",yaml:"statusCode,omitempty"`
	timeout      int    `json:"timeout,omitempty",yaml:"timeout,omitempty"`
	errorMessage string `json:"errorMessage,omitempty",yaml:"errorMessage,omitempty"`
}

func CreateConfig() *Config {
	return &Config{
		attempts:     1,
		Status:       make([]int, 100),
		timeout:      5,
		errorMessage: "Service Unavailable",
	}
}

type RetryIF struct {
	attempts     int
	next         http.Handler
	listener     Listener
	Status       []int
	timeout      int
	errorMessage string
}

func New(ctx context.Context, next http.Handler, config *Config) (http.Handler, error) {
	/*
		if len(config.Status) == 0 {
			return nil, fmt.Errorf("status is empty, please define at lease on statos code")
		}

	*/

	return &RetryIF{
		next:         next,
		attempts:     config.attempts,
		Status:       config.Status,
		timeout:      config.timeout,
		errorMessage: config.errorMessage,
	}, nil
}

func (retryIf *RetryIF) ServeHTTP(rw http.ResponseWriter, req *http.Request) {

	/*
		if retryIf.attempts == 1 {
			retryIf.next.ServeHTTP(rw, req)
			return
		}
	*/

	//attempts := 1
	fmt.Println("Hello From Plugin")

	retryIf.next.ServeHTTP(rw, req)
}

type Listener interface {
	Retried(attempts int)
}

func isCodeMatch(stCode int, listCodes []int) bool {
	var exists bool = false
	for _, code := range listCodes {
		if code == stCode {
			exists = true
		}
	}
	return exists
}
