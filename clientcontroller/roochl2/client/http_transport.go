package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// HttpHeaders represents HTTP header key-value pairs
type HttpHeaders map[string]string

// RoochHTTPTransportOptions contains configuration for the HTTP transport
type RoochHTTPTransportOptions struct {
	URL     string
	Headers HttpHeaders
	//RPC struct {
	//	Headers HttpHeaders
	//	URL     string
	//}
}

// RoochTransportRequestOptions represents the request parameters
type RoochTransportRequestOptions struct {
	Method string
	Params []interface{}
}

// jsonRPCRequest represents a JSON-RPC 2.0 request
type jsonRPCRequest struct {
	JsonRPC string        `json:"jsonrpc"`
	ID      int64         `json:"id"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
}

// jsonRPCResponse represents a JSON-RPC 2.0 response
type jsonRPCResponse struct {
	JsonRPC string `json:"jsonrpc"`
	ID      int64  `json:"id"`
	//Result  interface{}   `json:"result,omitempty"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *jsonRPCError   `json:"error,omitempty"`
}

// jsonRPCError represents a JSON-RPC 2.0 error
type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// RoochTransport defines the interface for making RPC requests
type RoochTransport interface {
	Request(method string, params []interface{}, result interface{}) error
}

// RoochHTTPTransport implements the HTTP transport layer for Rooch
type RoochHTTPTransport struct {
	options    RoochHTTPTransportOptions
	requestID  int64
	httpClient *http.Client
}

// NewRoochHTTPTransport creates a new RoochHTTPTransport instance
func NewRoochHTTPTransport(options RoochHTTPTransportOptions) *RoochHTTPTransport {
	httpClient := &http.Client{}
	// TODO Support timeout config
	httpClient.Timeout = 60 * time.Second

	return &RoochHTTPTransport{
		options:    options,
		httpClient: httpClient,
	}
}

// Request sends a JSON-RPC request and returns the response
// func (t *RoochHTTPTransport) Request(input RoochTransportRequestOptions) (interface{}, error) {
func (t *RoochHTTPTransport) Request(method string, params []interface{}, result interface{}) error {
	t.requestID++

	// Determine the URL to use
	url := t.options.URL
	//if t.options.RPC.URL != "" {
	//	url = t.options.RPC.URL
	//}

	// Create the JSON-RPC request
	reqBody := jsonRPCRequest{
		JsonRPC: "2.0",
		ID:      t.requestID,
		Method:  method,
		Params:  params,
	}

	// Marshal the request body
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create the HTTP request
	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	for key, value := range t.options.Headers {
		req.Header.Set(key, value)
	}

	// Send the request
	resp, err := t.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d %s", resp.StatusCode, resp.Status)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse response
	var jsonResp jsonRPCResponse
	if err := json.Unmarshal(body, &jsonResp); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Check for JSON-RPC error
	if jsonResp.Error != nil {
		return fmt.Errorf("JSON-RPC error: code=%d message=%s",
			jsonResp.Error.Code, jsonResp.Error.Message)
	}

	if err := json.Unmarshal(jsonResp.Result, result); err != nil {
		return fmt.Errorf("parse JSON-RPC result failed :%w", err)
	}

	return nil
}
