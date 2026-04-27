package server

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/neok/currency/internal/data"
	"github.com/neok/currency/internal/fetcher"
)

type stubStore struct {
	latest    []fetcher.Rate
	latestErr error
	history   []fetcher.Rate
	historyFn func(currency string, f data.HistoryFilter)
	calls     int
}

func (s *stubStore) Save(context.Context, fetcher.Rate) error { return nil }

func (s *stubStore) Latest(context.Context) ([]fetcher.Rate, error) {
	s.calls++
	return s.latest, s.latestErr
}

func (s *stubStore) History(_ context.Context, currency string, f data.HistoryFilter) ([]fetcher.Rate, error) {
	s.calls++
	if s.historyFn != nil {
		s.historyFn(currency, f)
	}
	return s.history, nil
}

type stubCache struct {
	mu      sync.Mutex
	entries map[string][]byte
}

func newStubCache() *stubCache { return &stubCache{entries: map[string][]byte{}} }

func (c *stubCache) Get(key string) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	v, ok := c.entries[key]
	return v, ok
}

func (c *stubCache) Set(key string, value []byte, _ time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[key] = value
}

func newHandler(store data.Store, cache *stubCache) *RatesHandler {
	return NewRatesHandler(store, cache, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func TestRatesHandler_Latest(t *testing.T) {
	store := &stubStore{latest: []fetcher.Rate{{Currency: "USD", Rate: 1.08, SourceDate: time.Now()}}}
	c := newStubCache()
	h := newHandler(store, c)

	req := httptest.NewRequest(http.MethodGet, "/v1/rates/latest", nil)
	rr := httptest.NewRecorder()
	h.Latest(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), `"USD"`) {
		t.Errorf("body missing USD: %s", rr.Body.String())
	}
	if _, ok := c.entries["latest"]; !ok {
		t.Error("expected response to be cached")
	}

	rr2 := httptest.NewRecorder()
	h.Latest(rr2, req)
	if store.calls != 1 {
		t.Errorf("store called %d times, want 1 (second call should hit cache)", store.calls)
	}
}

func TestRatesHandler_Latest_StoreError(t *testing.T) {
	store := &stubStore{latestErr: errors.New("db down")}
	h := newHandler(store, newStubCache())

	rr := httptest.NewRecorder()
	h.Latest(rr, httptest.NewRequest(http.MethodGet, "/v1/rates/latest", nil))

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rr.Code)
	}
}

func TestRatesHandler_History_Validation(t *testing.T) {
	tests := []struct {
		name     string
		currency string
		query    string
		wantKey  string
	}{
		{"bad currency", "usd", "", "currency"},
		{"bad limit", "USD", "?limit=0", "limit"},
		{"bad limit non-numeric", "USD", "?limit=abc", "limit"},
		{"bad order", "USD", "?order=sideways", "order"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newHandler(&stubStore{}, newStubCache())
			req := httptest.NewRequest(http.MethodGet, "/v1/rates/history/"+tt.currency+tt.query, nil)
			req.SetPathValue("currency", tt.currency)
			rr := httptest.NewRecorder()
			h.History(rr, req)

			if rr.Code != http.StatusUnprocessableEntity {
				t.Fatalf("status = %d, want 422", rr.Code)
			}
			var body struct {
				Errors map[string]string `json:"errors"`
			}
			if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if _, ok := body.Errors[tt.wantKey]; !ok {
				t.Errorf("expected error key %q, got %v", tt.wantKey, body.Errors)
			}
		})
	}
}

func TestRatesHandler_History_PassesFilter(t *testing.T) {
	var got data.HistoryFilter
	var gotCurrency string
	store := &stubStore{
		history: []fetcher.Rate{{Currency: "USD", Rate: 1.0, SourceDate: time.Now()}},
		historyFn: func(currency string, f data.HistoryFilter) {
			gotCurrency = currency
			got = f
		},
	}
	h := newHandler(store, newStubCache())

	req := httptest.NewRequest(http.MethodGet, "/v1/rates/history/USD?limit=10&order=asc", nil)
	req.SetPathValue("currency", "USD")
	rr := httptest.NewRecorder()
	h.History(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if gotCurrency != "USD" {
		t.Errorf("currency = %q, want USD", gotCurrency)
	}
	if got.Limit != 10 || got.Order != data.SortAsc {
		t.Errorf("filter = %+v, want {Limit:10 Order:asc}", got)
	}
}
