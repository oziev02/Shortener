package service

import (
	"crypto/rand"
	"encoding/base64"
	"strings"
)

// ShortenerService предоставляет методы для генерации коротких URL
type ShortenerService struct {
	baseURL string
}

// NewShortenerService создаёт новый экземпляр сервиса
func NewShortenerService(baseURL string) *ShortenerService {
	return &ShortenerService{baseURL: baseURL}
}

// GenerateShortURL генерирует случайный короткий URL
func (s *ShortenerService) GenerateShortURL(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	// Используем URL-safe base64 кодирование
	encoded := base64.URLEncoding.EncodeToString(bytes)
	// Убираем padding и ограничиваем длину
	encoded = strings.TrimRight(encoded, "=")
	if len(encoded) > length {
		encoded = encoded[:length]
	}

	return encoded, nil
}

// BuildShortURL строит полный короткий URL
func (s *ShortenerService) BuildShortURL(shortCode string) string {
	return s.baseURL + "/s/" + shortCode
}
