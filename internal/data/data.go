package data

import (
	"context"

	"github.com/neok/currency/internal/fetcher"
)

type SortOrder string

const (
	SortAsc  SortOrder = "asc"
	SortDesc SortOrder = "desc"
)

type HistoryFilter struct {
	Limit int
	Order SortOrder
}

type Store interface {
	Save(ctx context.Context, r fetcher.Rate) error
	Latest(ctx context.Context) ([]fetcher.Rate, error)
	History(ctx context.Context, currency string, f HistoryFilter) ([]fetcher.Rate, error)
}
