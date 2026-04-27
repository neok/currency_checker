package fetcher

import (
	"context"
	"log/slog"
	"strings"
	"time"
)

type HTTPFetcher struct {
	url       string
	transport Transport
	parser    RatesParser
	logger    *slog.Logger
}

func NewHTTPFetcher(url string, transport Transport, parser RatesParser, logger *slog.Logger) *HTTPFetcher {
	return &HTTPFetcher{url: url, transport: transport, parser: parser, logger: logger}
}

func (f *HTTPFetcher) FetchOne(ctx context.Context, currency string) (Rate, error) {
	currency = strings.ToUpper(currency)
	start := time.Now()

	f.logger.DebugContext(ctx, "fetcher: start", "currency", currency, "url", f.url)

	body, err := f.transport.Get(ctx, f.url)
	if err != nil {
		return Rate{}, err
	}
	defer body.Close()

	rate, err := f.parser.Parse(body, currency)
	if err != nil {
		return Rate{}, err
	}

	f.logger.DebugContext(ctx, "fetcher: done",
		"currency", currency,
		"rate", rate.Rate,
		"source_date", rate.SourceDate,
		"duration_ms", time.Since(start).Milliseconds(),
	)
	return rate, nil
}
