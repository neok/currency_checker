package job

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/neok/currency/internal/data"
	"github.com/neok/currency/internal/fetcher"
)

func Run(ctx context.Context, logger *slog.Logger, f fetcher.Fetcher, s data.Store, currencies []string) error {
	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		errs []error
	)

	for _, currency := range currencies {
		wg.Go(func() {
			// Per-currency timeout. Whole job deadline is the caller's responsibility.
			ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()

			rate, err := f.FetchOne(ctx, currency)
			if err == nil {
				err = s.Save(ctx, rate)
				if err == nil {
					logger.DebugContext(ctx, "rate saved", "currency", currency, "rate", rate.Rate, "source_date", rate.SourceDate)
				}
			}
			if err != nil {
				logger.ErrorContext(ctx, "fetch failed", "currency", currency, "err", err)
				mu.Lock()
				errs = append(errs, fmt.Errorf("%s: %w", currency, err))
				mu.Unlock()
			}
		})
	}
	wg.Wait()

	logger.InfoContext(ctx, "fetch complete",
		"ok", len(currencies)-len(errs),
		"failed", len(errs),
	)

	if len(errs) == len(currencies) {
		return fmt.Errorf("all fetches failed: %w", errors.Join(errs...))
	}
	return nil
}
