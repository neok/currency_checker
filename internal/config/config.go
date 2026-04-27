package config

import (
	"flag"
	"os"
	"strconv"
	"strings"
	"time"
)

//Config validation is intentionally skipped.
//Each value read from the environment below could be validated if needed

type DBConfig struct {
	DSN          string
	MaxOpenConns int
	MaxIdleConns int
	MaxIdleTime  time.Duration
}

type Base struct {
	LogLevel string
	DB       DBConfig
}

type APIConfig struct {
	Base
	HTTPAddr string
}

type FetchConfig struct {
	Base
	RSSURL     string
	Currencies []string
}

func LoadAPI() APIConfig {
	return APIConfig{
		Base:     loadBase(),
		HTTPAddr: os.Getenv("HTTP_ADDR"),
	}
}

func LoadFetch(args []string) (FetchConfig, error) {
	cfg := FetchConfig{Base: loadBase()}

	fs := flag.NewFlagSet("fetch", flag.ContinueOnError)
	fs.StringVar(&cfg.RSSURL, "rss-url", os.Getenv("RSS_URL"), "RSS feed URL")
	currencies := fs.String("currencies", os.Getenv("FETCH_CURRENCIES"), "comma-separated currency codes")

	if err := fs.Parse(args); err != nil {
		return FetchConfig{}, err
	}

	cfg.Currencies = parseCurrencies(*currencies)
	return cfg, nil
}

func loadBase() Base {
	maxOpen, _ := strconv.Atoi(os.Getenv("DB_MAX_OPEN"))
	maxIdle, _ := strconv.Atoi(os.Getenv("DB_MAX_IDLE"))
	maxIdleTime, _ := time.ParseDuration(os.Getenv("DB_MAX_IDLE_TIME"))

	return Base{
		LogLevel: os.Getenv("LOG_LEVEL"),
		DB: DBConfig{
			DSN:          os.Getenv("DB_DSN"),
			MaxOpenConns: maxOpen,
			MaxIdleConns: maxIdle,
			MaxIdleTime:  maxIdleTime,
		},
	}
}

func parseCurrencies(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if v := strings.TrimSpace(strings.ToUpper(p)); v != "" {
			out = append(out, v)
		}
	}
	return out
}
