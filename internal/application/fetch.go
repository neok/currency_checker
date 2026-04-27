package application

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/neok/currency/internal/config"
	"github.com/neok/currency/internal/data"
	"github.com/neok/currency/internal/fetcher"
)

type FetchApp struct {
	Config  config.FetchConfig
	Logger  *slog.Logger
	Store   data.Store
	Fetcher fetcher.Fetcher
}

func NewFetch(cfg config.FetchConfig) (*FetchApp, func(), error) {
	base, cleanup, err := newBaseApp(cfg.Base)
	if err != nil {
		return nil, nil, err
	}

	httpTransport := fetcher.NewHTTPTransport(&http.Client{Timeout: 10 * time.Second})
	parser := fetcher.NewBankRatesFeedParser()

	return &FetchApp{
		Config:  cfg,
		Logger:  base.logger,
		Store:   base.store,
		Fetcher: fetcher.NewHTTPFetcher(cfg.RSSURL, httpTransport, parser, base.logger),
	}, cleanup, nil
}
