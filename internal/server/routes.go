package server

import (
	"net/http"

	"github.com/neok/currency/internal/application"
)

func routes(app *application.APIApp) http.Handler {
	mux := http.NewServeMux()

	rates := NewRatesHandler(app.Store, app.ResponseCache, app.Logger)

	mux.HandleFunc("GET /v1/rates/latest", rates.Latest)
	mux.HandleFunc("GET /v1/rates/history/{currency}", rates.History)

	return logRequests(app, mux)
}
