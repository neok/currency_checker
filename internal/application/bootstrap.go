package application

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/neok/currency/internal/config"
	"github.com/neok/currency/internal/data"
)

type baseApp struct {
	logger *slog.Logger
	db     *sql.DB
	store  data.Store
}

func newBaseApp(base config.Base) (baseApp, func(), error) {
	logger, err := newLogger(base.LogLevel)
	if err != nil {
		return baseApp{}, nil, fmt.Errorf("logger: %w", err)
	}

	db, err := openDB(base.DB)
	if err != nil {
		return baseApp{}, nil, fmt.Errorf("open db: %w", err)
	}

	return baseApp{
		logger: logger,
		db:     db,
		store:  data.NewMySQLStore(db),
	}, func() { _ = db.Close() }, nil
}

func openDB(cfg config.DBConfig) (*sql.DB, error) {
	db, err := sql.Open("mysql", cfg.DSN)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxIdleTime(cfg.MaxIdleTime)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func newLogger(level string) (*slog.Logger, error) {
	var lvl slog.Level
	if err := lvl.UnmarshalText([]byte(level)); err != nil {
		return nil, err
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl})), nil
}
