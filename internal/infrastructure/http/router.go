package http

import (
	"net/http"
)

// Router настраивает маршруты
type Router struct {
	handler *Handler
}

// NewRouter создаёт новый роутер
func NewRouter(handler *Handler) *Router {
	return &Router{handler: handler}
}

// SetupRoutes настраивает все маршруты
func (r *Router) SetupRoutes() http.Handler {
	mux := http.NewServeMux()

	// API эндпоинты
	mux.HandleFunc("/shorten", r.handler.Shorten)
	mux.HandleFunc("/s/", r.handler.Redirect)
	mux.HandleFunc("/analytics/", r.handler.Analytics)

	// UI
	mux.HandleFunc("/", r.handler.ServeUI)

	return mux
}
