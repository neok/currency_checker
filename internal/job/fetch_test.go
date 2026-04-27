package job

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/neok/currency/internal/data"
	"github.com/neok/currency/internal/fetcher"
)

type stubFetcher struct {
	mu      sync.Mutex
	called  []string
	failFor map[string]bool
}

func (f *stubFetcher) FetchOne(_ context.Context, currency string) (fetcher.Rate, error) {
	f.mu.Lock()
	f.called = append(f.called, currency)
	f.mu.Unlock()
	if f.failFor[currency] {
		return fetcher.Rate{}, errors.New("upstream down")
	}
	return fetcher.Rate{Currency: currency, Rate: 1.0, SourceDate: time.Now()}, nil
}

type stubStore struct {
	saved atomic.Int32
}

func (s *stubStore) Save(context.Context, fetcher.Rate) error      { s.saved.Add(1); return nil }
func (s *stubStore) Latest(context.Context) ([]fetcher.Rate, error) { return nil, nil }
func (s *stubStore) History(context.Context, string, data.HistoryFilter) ([]fetcher.Rate, error) {
	return nil, nil
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestRun(t *testing.T) {
	tests := []struct {
		name        string
		currencies  []string
		failFor     map[string]bool
		wantSaved   int32
		wantErr     bool
	}{
		{
			name:       "all succeed",
			currencies: []string{"USD", "GBP", "EUR"},
			wantSaved:  3,
		},
		{
			name:       "partial failure returns nil",
			currencies: []string{"USD", "GBP", "EUR"},
			failFor:    map[string]bool{"GBP": true},
			wantSaved:  2,
		},
		{
			name:       "all fail returns error",
			currencies: []string{"USD", "GBP"},
			failFor:    map[string]bool{"USD": true, "GBP": true},
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &stubFetcher{failFor: tt.failFor}
			s := &stubStore{}

			err := Run(context.Background(), discardLogger(), f, s, tt.currencies)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr = %v", err, tt.wantErr)
			}
			if s.saved.Load() != tt.wantSaved {
				t.Errorf("saved = %d, want %d", s.saved.Load(), tt.wantSaved)
			}
			if len(f.called) != len(tt.currencies) {
				t.Errorf("fetcher called %d times, want %d", len(f.called), len(tt.currencies))
			}
		})
	}
}
