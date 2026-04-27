package application

import (
	"database/sql"
	"log/slog"

	"github.com/neok/currency/internal/cache"
	"github.com/neok/currency/internal/config"
	"github.com/neok/currency/internal/data"
)

type APIApp struct {
	Config        config.APIConfig
	Logger        *slog.Logger
	DB            *sql.DB
	Store         data.Store
	ResponseCache cache.Cache[[]byte]
}

func NewAPI(cfg config.APIConfig) (*APIApp, func(), error) {
	base, cleanup, err := newBaseApp(cfg.Base)
	if err != nil {
		return nil, nil, err
	}
	return &APIApp{
		Config:        cfg,
		Logger:        base.logger,
		DB:            base.db,
		Store:         base.store,
		ResponseCache: cache.NewInMemoryCache[[]byte](),
	}, cleanup, nil
}
