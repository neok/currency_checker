package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/neok/currency/internal/cache"
	"github.com/neok/currency/internal/data"
	"github.com/neok/currency/internal/fetcher"
	"github.com/neok/currency/internal/validator"
)

const responseCacheTTL = 5 * time.Minute

var currencyRE = regexp.MustCompile(`^[A-Z]{3}$`)

type rateDTO struct {
	Currency   string    `json:"currency"`
	Rate       float64   `json:"rate"`
	SourceDate time.Time `json:"source_date"`
}

type RatesHandler struct {
	store  data.Store
	cache  cache.Cache[[]byte]
	logger *slog.Logger
}

func NewRatesHandler(store data.Store, c cache.Cache[[]byte], logger *slog.Logger) *RatesHandler {
	return &RatesHandler{store: store, cache: c, logger: logger}
}

func (h *RatesHandler) Latest(w http.ResponseWriter, r *http.Request) {
	h.serveCachedRates(w, r, "latest", "latest", h.store.Latest)
}

func (h *RatesHandler) History(w http.ResponseWriter, r *http.Request) {
	currency := r.PathValue("currency")
	v := validator.New()
	filter := h.validateHistory(currency, r.URL.Query(), v)

	if !v.Valid() {
		h.logger.DebugContext(r.Context(), "history validation failed", "currency", currency, "errors", v.Errors)
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"errors": v.Errors})
		return
	}

	h.logger.DebugContext(r.Context(), "history requested", "currency", currency, "limit", filter.Limit, "order", filter.Order)
	key := "history:" + currency + ":" + string(filter.Order) + ":" + strconv.Itoa(filter.Limit)
	h.serveCachedRates(w, r, key, "history", func(ctx context.Context) ([]fetcher.Rate, error) {
		return h.store.History(ctx, currency, filter)
	})
}

func (h *RatesHandler) validateHistory(currency string, q url.Values, v *validator.Validator) data.HistoryFilter {
	v.Check(validator.Matches(currency, currencyRE), "currency", "must be a 3-letter uppercase code")

	filter := data.HistoryFilter{Order: data.SortDesc}
	if s := q.Get("limit"); s != "" {
		n, err := strconv.Atoi(s)
		v.Check(err == nil && n >= 1, "limit", "must be a positive integer")
		filter.Limit = n
	}
	if s := q.Get("order"); s != "" {
		order := data.SortOrder(strings.ToLower(s))
		v.Check(validator.In(order, data.SortAsc, data.SortDesc), "order", "must be 'asc' or 'desc'")
		filter.Order = order
	}
	return filter
}

func (h *RatesHandler) serveCachedRates(w http.ResponseWriter, r *http.Request, key, op string, fetch func(context.Context) ([]fetcher.Rate, error)) {
	w.Header().Set("Cache-Control", "public, max-age="+strconv.Itoa(int(responseCacheTTL.Seconds())))

	if body, ok := h.cache.Get(key); ok {
		h.logger.DebugContext(r.Context(), "cache hit", "op", op, "key", key, "bytes", len(body))
		writeRaw(w, http.StatusOK, body)
		return
	}
	h.logger.DebugContext(r.Context(), "cache miss", "op", op, "key", key)

	rates, err := fetch(r.Context())
	if err != nil {
		h.logger.ErrorContext(r.Context(), op+" failed", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	body, err := json.Marshal(convertToDTOs(rates))
	if err != nil {
		h.logger.ErrorContext(r.Context(), "marshal failed", "op", op, "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	h.cache.Set(key, body, responseCacheTTL)
	h.logger.DebugContext(r.Context(), "cache set", "op", op, "key", key, "rows", len(rates), "bytes", len(body))
	writeRaw(w, http.StatusOK, body)
}

func convertToDTOs(rates []fetcher.Rate) []rateDTO {
	out := make([]rateDTO, len(rates))
	for i, r := range rates {
		out[i] = rateDTO{Currency: r.Currency, Rate: r.Rate, SourceDate: r.SourceDate}
	}
	return out
}
