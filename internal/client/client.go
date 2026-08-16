package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const maxErrorBodyBytes = 64 << 10

const (
	invalidProjectID = 401
	projectNotFound  = 403
	invalidQueueID   = 701
)

// ClearMLClient is a minimal client for the ClearML REST API endpoints used by
// this provider. It deliberately does not retry mutations: a timed-out POST
// cannot be assumed not to have reached ClearML.
type ClearMLClient struct {
	userAgent string
	apiToken  string
	apiURL    *url.URL
	http      *http.Client
}

// APIError is returned when ClearML responds with a non-success status.
type APIError struct {
	StatusCode    int
	ResultCode    int
	ResultSubcode int
	Method        string
	Path          string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("ClearML API %s %s returned HTTP %d: %s", e.Method, e.Path, e.StatusCode, http.StatusText(e.StatusCode))
}

func NewClearMLClient(ctx context.Context, userAgent, accessKey, secretKey, apiURL string) (*ClearMLClient, error) {
	u, err := url.Parse(apiURL)
	if err != nil {
		return nil, fmt.Errorf("parse ClearML API URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("ClearML API URL must use http or https")
	}
	if u.Host == "" {
		return nil, fmt.Errorf("ClearML API URL must include a host")
	}

	c := &ClearMLClient{
		userAgent: userAgent,
		apiURL:    u.ResolveReference(&url.URL{Path: "/"}),
		http:      &http.Client{Timeout: 30 * time.Second},
	}

	token, err := c.getAPIToken(ctx, accessKey, secretKey)
	if err != nil {
		return nil, err
	}
	c.apiToken = token
	return c, nil
}

func (c *ClearMLClient) getAPIToken(ctx context.Context, accessKey, secretKey string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpointURL("/auth.login", nil), nil)
	if err != nil {
		return "", fmt.Errorf("create ClearML login request: %w", err)
	}
	req.SetBasicAuth(accessKey, secretKey)
	req.Header.Set("User-Agent", c.userAgent)

	body, err := c.do(req, "/auth.login")
	if err != nil {
		return "", err
	}

	var response struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("decode ClearML login response: %w", err)
	}
	if response.Data.Token == "" {
		return "", errors.New("ClearML login response did not contain a token")
	}
	return response.Data.Token, nil
}

func (c *ClearMLClient) request(ctx context.Context, method, path string, payload any, output any) error {
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("encode ClearML request payload: %w", err)
		}
		body = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.endpointURL(path, nil), body)
	if err != nil {
		return fmt.Errorf("create ClearML request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiToken)
	req.Header.Set("User-Agent", c.userAgent)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	responseBody, err := c.do(req, path)
	if err != nil {
		return err
	}
	if output == nil || len(responseBody) == 0 {
		return nil
	}
	if err := json.Unmarshal(responseBody, output); err != nil {
		return fmt.Errorf("decode ClearML response for %s: %w", path, err)
	}
	return nil
}

func (c *ClearMLClient) endpointURL(path string, query url.Values) string {
	u := c.apiURL.ResolveReference(&url.URL{Path: path})
	u.RawQuery = query.Encode()
	return u.String()
}

func (c *ClearMLClient) do(req *http.Request, path string) ([]byte, error) {
	response, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send ClearML request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode >= http.StatusBadRequest {
		body, readErr := io.ReadAll(io.LimitReader(response.Body, maxErrorBodyBytes))
		if readErr != nil {
			return nil, fmt.Errorf("read ClearML error response: %w", readErr)
		}
		var envelope struct {
			Meta struct {
				ResultCode    int `json:"result_code"`
				ResultSubcode int `json:"result_subcode"`
			} `json:"meta"`
		}
		_ = json.Unmarshal(body, &envelope)
		return nil, &APIError{
			StatusCode: response.StatusCode, ResultCode: envelope.Meta.ResultCode,
			ResultSubcode: envelope.Meta.ResultSubcode, Method: req.Method, Path: path,
		}
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("read ClearML response: %w", err)
	}
	return body, nil
}

func IsNotFound(err error) bool {
	var notFound *NotFoundError
	if errors.As(err, &notFound) {
		return true
	}
	var apiError *APIError
	if !errors.As(err, &apiError) {
		return false
	}
	if apiError.StatusCode == http.StatusNotFound {
		return true
	}
	if apiError.ResultCode != http.StatusBadRequest {
		return false
	}
	switch apiError.ResultSubcode {
	case invalidProjectID, projectNotFound, invalidQueueID:
		return true
	default:
		return false
	}
}
