package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/oziev02/Shortener/internal/application/usecase"
)

const (
	// MaxURLLength максимальная длина URL
	MaxURLLength = 2048
	// MaxRequestBodySize максимальный размер тела запроса (1MB)
	MaxRequestBodySize = 1024 * 1024
)

// ErrorResponse структурированный ответ об ошибке
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

// Handler обрабатывает HTTP запросы
type Handler struct {
	shortenUseCase   *usecase.ShortenUseCase
	redirectUseCase  *usecase.RedirectUseCase
	analyticsUseCase *usecase.AnalyticsUseCase
	logger           Logger
}

// NewHandler создаёт новый HTTP handler
func NewHandler(
	shortenUseCase *usecase.ShortenUseCase,
	redirectUseCase *usecase.RedirectUseCase,
	analyticsUseCase *usecase.AnalyticsUseCase,
	logger Logger,
) *Handler {
	if logger == nil {
		logger = &NoOpLogger{}
	}
	return &Handler{
		shortenUseCase:   shortenUseCase,
		redirectUseCase:  redirectUseCase,
		analyticsUseCase: analyticsUseCase,
		logger:           logger,
	}
}

// Shorten обрабатывает POST /shorten
func (h *Handler) Shorten(w http.ResponseWriter, r *http.Request) {
	if !h.ensureMethod(w, r, http.MethodPost) {
		return
	}

	// Ограничиваем размер тела запроса
	r.Body = http.MaxBytesReader(w, r.Body, MaxRequestBodySize)
	defer r.Body.Close()

	var req usecase.CreateLinkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid_request_body", "Invalid request body", err)
		return
	}

	// Валидация URL
	if req.OriginalURL == "" {
		h.respondError(w, http.StatusBadRequest, "url_required", "original_url is required", nil)
		return
	}

	if err := validateURL(req.OriginalURL); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid_url", "Invalid URL format", err)
		return
	}

	resp, err := h.shortenUseCase.Execute(r.Context(), req)
	if err != nil {
		h.handleUseCaseError(w, err)
		return
	}

	h.respondJSON(w, http.StatusCreated, resp)
}

// Redirect обрабатывает GET /s/{short_url}
func (h *Handler) Redirect(w http.ResponseWriter, r *http.Request) {
	if !h.ensureMethod(w, r, http.MethodGet) {
		return
	}

	shortURL, ok := h.extractPathParam(r, "/s/")
	if !ok {
		h.respondError(w, http.StatusBadRequest, "invalid_short_url", "Invalid short URL", nil)
		return
	}

	userAgent := r.Header.Get("User-Agent")
	ipAddress := getIPAddress(r)

	originalURL, err := h.redirectUseCase.Execute(r.Context(), shortURL, userAgent, ipAddress)
	if err != nil {
		h.handleUseCaseError(w, err)
		return
	}

	http.Redirect(w, r, originalURL, http.StatusFound)
}

// Analytics обрабатывает GET /analytics/{short_url}
func (h *Handler) Analytics(w http.ResponseWriter, r *http.Request) {
	if !h.ensureMethod(w, r, http.MethodGet) {
		return
	}

	shortURL, ok := h.extractPathParam(r, "/analytics/")
	if !ok {
		h.respondError(w, http.StatusBadRequest, "invalid_short_url", "Invalid short URL", nil)
		return
	}

	analytics, err := h.analyticsUseCase.Execute(r.Context(), shortURL)
	if err != nil {
		h.handleUseCaseError(w, err)
		return
	}

	h.respondJSON(w, http.StatusOK, analytics)
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

// ensureMethod проверяет HTTP метод и возвращает false если метод неверный
func (h *Handler) ensureMethod(w http.ResponseWriter, r *http.Request, allowedMethod string) bool {
	if r.Method != allowedMethod {
		h.respondError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed", nil)
		return false
	}
	return true
}

// extractPathParam извлекает параметр из пути URL
func (h *Handler) extractPathParam(r *http.Request, prefix string) (string, bool) {
	path := strings.TrimPrefix(r.URL.Path, prefix)
	if path == "" || path == r.URL.Path {
		return "", false
	}
	return path, true
}

// handleUseCaseError обрабатывает ошибки use case слоя
func (h *Handler) handleUseCaseError(w http.ResponseWriter, err error) {
	h.logger.Error("usecase error", err)

	switch {
	case errors.Is(err, usecase.ErrAliasExists):
		h.respondError(w, http.StatusConflict, "alias_exists", err.Error(), err)
	case errors.Is(err, usecase.ErrLinkNotFound):
		h.respondError(w, http.StatusNotFound, "link_not_found", "Link not found", err)
	case errors.Is(err, usecase.ErrInvalidURL):
		h.respondError(w, http.StatusBadRequest, "invalid_url", err.Error(), err)
	case errors.Is(err, usecase.ErrURLRequired):
		h.respondError(w, http.StatusBadRequest, "url_required", err.Error(), err)
	default:
		h.respondError(w, http.StatusInternalServerError, "internal_error", "Internal server error", err)
	}
}

// respondError отправляет структурированный ответ об ошибке
func (h *Handler) respondError(w http.ResponseWriter, statusCode int, code, message string, err error) {
	if err != nil {
		h.logger.Error(message, err, "code", code, "status", statusCode)
	}

	response := ErrorResponse{
		Error:   message,
		Code:    code,
		Message: message,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if encodeErr := json.NewEncoder(w).Encode(response); encodeErr != nil {
		h.logger.Error("failed to encode error response", encodeErr)
	}
}

// respondJSON отправляет JSON ответ
func (h *Handler) respondJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("failed to encode JSON response", err)
	}
}

// validateURL проверяет валидность URL
func validateURL(urlStr string) error {
	if len(urlStr) > MaxURLLength {
		return errors.New("URL too long")
	}

	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return err
	}

	if parsedURL.Scheme == "" {
		return errors.New("URL scheme is required")
	}

	if parsedURL.Host == "" {
		return errors.New("URL host is required")
	}

	// Проверяем что схема http или https
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return errors.New("URL scheme must be http or https")
	}

	return nil
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
