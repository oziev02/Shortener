package http

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/oziev02/Shortener/internal/application/usecase"
)

// Handler обрабатывает HTTP запросы
type Handler struct {
	shortenUseCase   *usecase.ShortenUseCase
	redirectUseCase  *usecase.RedirectUseCase
	analyticsUseCase *usecase.AnalyticsUseCase
}

// NewHandler создаёт новый HTTP handler
func NewHandler(
	shortenUseCase *usecase.ShortenUseCase,
	redirectUseCase *usecase.RedirectUseCase,
	analyticsUseCase *usecase.AnalyticsUseCase,
) *Handler {
	return &Handler{
		shortenUseCase:   shortenUseCase,
		redirectUseCase:  redirectUseCase,
		analyticsUseCase: analyticsUseCase,
	}
}

// Shorten обрабатывает POST /shorten
func (h *Handler) Shorten(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req usecase.CreateLinkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.OriginalURL == "" {
		http.Error(w, "original_url is required", http.StatusBadRequest)
		return
	}

	resp, err := h.shortenUseCase.Execute(r.Context(), req)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

// Redirect обрабатывает GET /s/{short_url}
func (h *Handler) Redirect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Извлекаем short_url из пути
	path := strings.TrimPrefix(r.URL.Path, "/s/")
	if path == "" || path == r.URL.Path {
		http.Error(w, "Invalid short URL", http.StatusBadRequest)
		return
	}

	userAgent := r.Header.Get("User-Agent")
	ipAddress := getIPAddress(r)

	originalURL, err := h.redirectUseCase.Execute(r.Context(), path, userAgent, ipAddress)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Link not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, originalURL, http.StatusFound)
}

// Analytics обрабатывает GET /analytics/{short_url}
func (h *Handler) Analytics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Извлекаем short_url из пути
	path := strings.TrimPrefix(r.URL.Path, "/analytics/")
	if path == "" || path == r.URL.Path {
		http.Error(w, "Invalid short URL", http.StatusBadRequest)
		return
	}

	analytics, err := h.analyticsUseCase.Execute(r.Context(), path)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Link not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(analytics)
}

// ServeUI обрабатывает запросы к UI
func (h *Handler) ServeUI(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" || r.URL.Path == "/index.html" {
		http.ServeFile(w, r, "./web/index.html")
		return
	}

	// Статические файлы
	if strings.HasPrefix(r.URL.Path, "/static/") {
		http.StripPrefix("/static/", http.FileServer(http.Dir("./web/static"))).ServeHTTP(w, r)
		return
	}

	http.NotFound(w, r)
}

// getIPAddress извлекает IP адрес из запроса
func getIPAddress(r *http.Request) string {
	// Проверяем заголовок X-Forwarded-For (для прокси)
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		ips := strings.Split(forwarded, ",")
		return strings.TrimSpace(ips[0])
	}

	// Проверяем заголовок X-Real-IP
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}

	// Используем RemoteAddr
	ip := r.RemoteAddr
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	return ip
}
