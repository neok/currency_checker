package data

import (
	"context"
	"database/sql"
	"strings"

	"github.com/neok/currency/internal/fetcher"
)

type MySQLStore struct {
	db *sql.DB
}

func NewMySQLStore(db *sql.DB) *MySQLStore {
	return &MySQLStore{db: db}
}

func (s *MySQLStore) Save(ctx context.Context, r fetcher.Rate) error {
	const q = `
		INSERT INTO exchange_rates (currency, rate, source_date)
		VALUES (?, ?, ?)
		ON DUPLICATE KEY UPDATE rate = VALUES(rate), fetched_at = CURRENT_TIMESTAMP(3)
	`
	_, err := s.db.ExecContext(ctx, q, r.Currency, r.Rate, r.SourceDate)
	return err
}

func (s *MySQLStore) Latest(ctx context.Context) ([]fetcher.Rate, error) {
	const q = `
		SELECT er.currency, er.rate, er.source_date
		FROM exchange_rates er
		INNER JOIN (
			SELECT currency, MAX(source_date) AS max_date
			FROM exchange_rates
			GROUP BY currency
		) m ON m.currency = er.currency AND m.max_date = er.source_date
		ORDER BY er.currency
	`
	rows, err := s.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []fetcher.Rate
	for rows.Next() {
		var r fetcher.Rate
		if err := rows.Scan(&r.Currency, &r.Rate, &r.SourceDate); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *MySQLStore) History(ctx context.Context, currency string, f HistoryFilter) ([]fetcher.Rate, error) {
	currency = strings.ToUpper(currency)

	limit := f.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}

	order := "DESC"
	if f.Order == SortAsc {
		order = "ASC"
	}

	q := `
		SELECT currency, rate, source_date
		FROM exchange_rates
		WHERE currency = ?
		ORDER BY source_date ` + order + `
		LIMIT ?
	`

	rows, err := s.db.QueryContext(ctx, q, currency, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []fetcher.Rate
	for rows.Next() {
		var r fetcher.Rate
		if err := rows.Scan(&r.Currency, &r.Rate, &r.SourceDate); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
