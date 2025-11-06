package simulator

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-resty/resty/v2"
)

// Request represents an HTTP request to simulate
type Request struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
	Body    interface{}       `json:"body"`
	Timeout int               `json:"timeout"` // in seconds
}

// Response represents an HTTP response
type Response struct {
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
	TimeTaken  time.Duration     `json:"time_taken"`
}

// Client represents a request simulator client
type Client struct {
	client *resty.Client
}

// New creates a new simulator client
func New() *Client {
	return &Client{
		client: resty.New(),
	}
}

// Simulate sends an HTTP request and returns the response
func (c *Client) Simulate(req *Request) (*Response, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}

	// Set timeout
	timeout := 30 * time.Second
	if req.Timeout > 0 {
		timeout = time.Duration(req.Timeout) * time.Second
	}
	c.client.SetTimeout(timeout)

	// Prepare request
	restyReq := c.client.R()

	// Set headers
	if req.Headers != nil {
		restyReq.SetHeaders(req.Headers)
	}

	// Set body
	if req.Body != nil {
		// If body is a string, set it directly
		if bodyStr, ok := req.Body.(string); ok {
			restyReq.SetBody(bodyStr)
		} else {
			// Otherwise, marshal it to JSON
			bodyBytes, err := json.Marshal(req.Body)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal request body: %w", err)
			}
			restyReq.SetBody(bodyBytes)
		}
	}

	// Record start time
	startTime := time.Now()

	// Send request based on method
	var resp *resty.Response
	var err error

	switch req.Method {
	case "GET", "":
		resp, err = restyReq.Get(req.URL)
	case "POST":
		resp, err = restyReq.Post(req.URL)
	case "PUT":
		resp, err = restyReq.Put(req.URL)
	case "DELETE":
		resp, err = restyReq.Delete(req.URL)
	case "PATCH":
		resp, err = restyReq.Patch(req.URL)
	case "HEAD":
		resp, err = restyReq.Head(req.URL)
	default:
		return nil, fmt.Errorf("unsupported HTTP method: %s", req.Method)
	}

	// Calculate time taken
	timeTaken := time.Since(startTime)

	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// Prepare response headers
	headers := make(map[string]string)
	for key, values := range resp.Header() {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}

	// Create response
	response := &Response{
		StatusCode: resp.StatusCode(),
		Headers:    headers,
		Body:       string(resp.Body()),
		TimeTaken:  timeTaken,
	}

	return response, nil
}

// SimulateWithValidation sends an HTTP request and validates the response
func (c *Client) SimulateWithValidation(req *Request, expectedStatusCode int) (*Response, error) {
	resp, err := c.Simulate(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != expectedStatusCode {
		return resp, fmt.Errorf("expected status code %d, got %d", expectedStatusCode, resp.StatusCode)
	}

	return resp, nil
}

// BatchSimulate sends multiple requests concurrently
func (c *Client) BatchSimulate(requests []*Request) ([]*Response, []error) {
	// Create channels for results and errors
	responses := make([]*Response, len(requests))
	errors := make([]error, len(requests))

	// Create a channel to signal completion
	done := make(chan bool, len(requests))

	// Send requests concurrently
	for i, req := range requests {
		go func(index int, request *Request) {
			defer func() { done <- true }()

			resp, err := c.Simulate(request)
			responses[index] = resp
			errors[index] = err
		}(i, req)
	}

	// Wait for all requests to complete
	for i := 0; i < len(requests); i++ {
		<-done
	}

	return responses, errors
}

// CreateRequestFromSwaggerOperation creates a request based on a Swagger operation
func (c *Client) CreateRequestFromSwaggerOperation(baseURL, path string, operation map[string]interface{}) (*Request, error) {
	// This is a simplified implementation
	// In a real-world scenario, you would parse the Swagger operation details

	req := &Request{
		Method:  "GET", // default
		URL:     baseURL + path,
		Headers: make(map[string]string),
	}

	// Try to determine method from operation keys
	for method := range operation {
		// Convert to uppercase to match HTTP methods
		methodUpper := fmt.Sprintf("%s", method)
		switch methodUpper {
		case "get", "post", "put", "delete", "patch", "head", "options":
			req.Method = methodUpper
		}
	}

	// Set default headers for JSON APIs
	req.Headers["Content-Type"] = "application/json"
	req.Headers["Accept"] = "application/json"

	return req, nil
}