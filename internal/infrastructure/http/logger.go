package http

import "log"

// Logger интерфейс для логирования
type Logger interface {
	Error(msg string, err error, fields ...interface{})
	Info(msg string, fields ...interface{})
	Debug(msg string, fields ...interface{})
}

// NoOpLogger пустая реализация логгера (для тестов или когда логирование не нужно)
type NoOpLogger struct{}

func (n *NoOpLogger) Error(msg string, err error, fields ...interface{}) {}
func (n *NoOpLogger) Info(msg string, fields ...interface{})             {}
func (n *NoOpLogger) Debug(msg string, fields ...interface{})            {}

// StdLogger простая реализация логгера используя стандартный log пакет
type StdLogger struct {
	logger *log.Logger
}

// NewStdLogger создаёт новый стандартный логгер
func NewStdLogger() *StdLogger {
	return &StdLogger{
		logger: log.Default(),
	}
}

func (s *StdLogger) Error(msg string, err error, fields ...interface{}) {
	if err != nil {
		// В реальном приложении используйте структурированное логирование (zap, logrus и т.д.)
		s.logger.Printf("ERROR: %s, error: %v", msg, err)
	} else {
		s.logger.Printf("ERROR: %s", msg)
	}
}

func (s *StdLogger) Info(msg string, fields ...interface{}) {
	s.logger.Printf("INFO: %s", msg)
}

func (s *StdLogger) Debug(msg string, fields ...interface{}) {
	s.logger.Printf("DEBUG: %s", msg)
}
