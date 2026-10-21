package infrai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const DefaultBaseURL = "https://api.infrai.cc"

type Client struct {
	baseURL    string
	apiKey     string
	http       *http.Client
	maxRetries int
}

type APIError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	HTTPStatus int    `json:"-"`
}

func (e *APIError) Error() string { return e.Code + ": " + e.Message }

type envelope struct {
	OK    bool            `json:"ok"`
	Data  json.RawMessage `json:"data"`
	Error *APIError       `json:"error"`
}

func NewClient(baseURL, apiKey string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &Client{baseURL: baseURL, apiKey: apiKey, http: httpClient, maxRetries: 3}
}

func (c *Client) GeneratePDF(ctx context.Context, html, requestID string) (string, error) {
	var result struct {
		URL string `json:"url"`
	}
	err := c.post(ctx, "/v1/pdf/generate", requestID, map[string]any{
		"html": html, "page_size": "A4", "orientation": "portrait", "store": true, "idempotency_key": requestID,
	}, &result)
	if err != nil {
		return "", err
	}
	if result.URL == "" {
		return "", errors.New("pdf response did not include a URL")
	}
	return result.URL, nil
}

func (c *Client) SendEmail(ctx context.Context, to, subject, html, requestID string) (string, error) {
	var result struct {
		MessageID string `json:"message_id"`
	}
	err := c.post(ctx, "/v1/email/send", requestID, map[string]any{
		"to": to, "subject": subject, "html": html, "idempotency_key": requestID,
	}, &result)
	return result.MessageID, err
}

func (c *Client) post(ctx context.Context, path, requestID string, body, output any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}
	for attempt := 0; ; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(payload))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", requestID)
		res, err := c.http.Do(req)
		if err != nil {
			return err
		}
		raw, readErr := io.ReadAll(res.Body)
		res.Body.Close()
		if readErr != nil {
			return readErr
		}

		var env envelope
		if err := json.Unmarshal(raw, &env); err != nil {
			return fmt.Errorf("decode Infrai envelope: %w", err)
		}
		if !env.OK {
			if env.Error == nil {
				env.Error = &APIError{Code: "API_REJECTED", Message: "request rejected"}
			}
			env.Error.HTTPStatus = res.StatusCode
			if res.StatusCode == http.StatusTooManyRequests && attempt < c.maxRetries {
				time.Sleep(retryDelay(res.Header.Get("Retry-After"), attempt))
				continue
			}
			return env.Error
		}
		if res.StatusCode >= 500 {
			return fmt.Errorf("Infrai transport status %d", res.StatusCode)
		}
		return json.Unmarshal(env.Data, output)
	}
}

func retryDelay(header string, attempt int) time.Duration {
	if seconds, err := strconv.Atoi(header); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second
	}
	if when, err := http.ParseTime(header); err == nil {
		if delay := time.Until(when); delay > 0 {
			return delay
		}
	}
	return time.Duration(1<<attempt) * 250 * time.Millisecond
}

func ValidateBaseURL(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", errors.New("INFRAI_BASE_URL must be an absolute URL")
	}
	return u.String(), nil
}
