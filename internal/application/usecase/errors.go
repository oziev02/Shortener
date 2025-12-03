package usecase

import "errors"

// Ошибки use case слоя
var (
	// ErrAliasExists возвращается когда кастомный алиас уже существует
	ErrAliasExists = errors.New("custom alias already exists")

	// ErrLinkNotFound возвращается когда ссылка не найдена
	ErrLinkNotFound = errors.New("link not found")

	// ErrInvalidURL возвращается когда URL имеет неверный формат
	ErrInvalidURL = errors.New("invalid URL format")

	// ErrURLRequired возвращается когда URL не указан
	ErrURLRequired = errors.New("original_url is required")
)
