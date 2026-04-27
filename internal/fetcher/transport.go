package fetcher

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

type Transport interface {
	Get(ctx context.Context, url string) (io.ReadCloser, error)
}

type HTTPTransport struct {
	client *http.Client
}

func NewHTTPTransport(client *http.Client) *HTTPTransport {
	return &HTTPTransport{client: client}
}

func (t *HTTPTransport) Get(ctx context.Context, url string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	resp, err := t.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http get: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("unexpected status: %s", resp.Status)
	}
	return resp.Body, nil
}
