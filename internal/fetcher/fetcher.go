package fetcher

import (
	"context"
	"io"
	"time"
)

type Rate struct {
	Currency   string
	Rate       float64
	SourceDate time.Time
}

type Fetcher interface {
	FetchOne(ctx context.Context, currency string) (Rate, error)
}

type RatesParser interface {
	Parse(body io.Reader, currency string) (Rate, error)
}
