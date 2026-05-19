package testingutils

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
)

type RequestOption func(*http.Request)

// helper for setting custom request id
func WithRequestID(requestID string) RequestOption {
	return func(req *http.Request) {
		req.Header.Set("X-Request-Id", requestID)
	}
}

// helper for setting auth token
func WithAuth(token string) RequestOption {
	return func(req *http.Request) {
		req.Header.Set("Authorization", "Bearer "+token)
	}
}

func MakeJSONRequest(method, path string, body interface{}, opts ...RequestOption) *http.Request {
	var bodyReader io.Reader
	if body != nil {
		if b, ok := body.([]byte); ok {
			bodyReader = bytes.NewBuffer(b)
		} else {
			marshaled, _ := json.Marshal(body)
			bodyReader = bytes.NewBuffer(marshaled)
		}
	}

	req, _ := http.NewRequest(method, path, bodyReader)
	req.Host = "localhost"
	req.Header.Set("Content-Type", "application/json")

	req.Header.Set("X-Request-Id", "TEST-REQUEST-ID")

	for _, opt := range opts {
		opt(req)
	}

	return req
}