package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/oziev02/Shortener/internal/application/usecase"
	"github.com/oziev02/Shortener/internal/config"
	"github.com/oziev02/Shortener/internal/domain/service"
	"github.com/oziev02/Shortener/internal/infrastructure/cache"
	"github.com/oziev02/Shortener/internal/infrastructure/database"
	httphandler "github.com/oziev02/Shortener/internal/infrastructure/http"
)

func main() {
	// Загружаем конфигурацию из переменных окружения и .env файла
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Параметры командной строки (имеют приоритет над переменными окружения)
	// Значения по умолчанию берутся из конфигурации
	port := flag.String("port", cfg.Port, "Server port")
	dbDSN := flag.String("db", cfg.DatabaseDSN, "Database DSN")
	redisAddr := flag.String("redis", cfg.RedisAddr, "Redis address")
	redisPassword := flag.String("redis-password", cfg.RedisPassword, "Redis password")
	baseURL := flag.String("base-url", cfg.BaseURL, "Base URL for short links")
	enableRedis := flag.Bool("enable-redis", cfg.EnableRedis, "Enable Redis caching")
	flag.Parse()

	// Подключение к БД
	db, err := database.NewPostgresDB(*dbDSN)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Подключение к Redis (опционально)
	var redisCache *cache.RedisCache
	if *enableRedis {
		redisCache, err = cache.NewRedisCache(*redisAddr, *redisPassword, 0, cfg.RedisTTL)
		if err != nil {
			log.Printf("Warning: Failed to connect to Redis: %v. Continuing without cache.", err)
		} else {
			defer redisCache.Close()
			log.Println("Redis cache enabled")
		}
	}

	// Инициализация репозиториев
	linkRepo := database.NewLinkRepository(db)
	clickRepo := database.NewClickRepository(db)

	// Инициализация сервисов
	shortenerService := service.NewShortenerService(*baseURL)

	// Инициализация use cases
	shortenUC := usecase.NewShortenUseCase(linkRepo, shortenerService, redisCache)
	redirectUC := usecase.NewRedirectUseCase(linkRepo, clickRepo, redisCache)
	analyticsUC := usecase.NewAnalyticsUseCase(linkRepo, clickRepo, redisCache)

	// Инициализация HTTP handler с логгером
	logger := httphandler.NewStdLogger()
	handler := httphandler.NewHandler(shortenUC, redirectUC, analyticsUC, logger)
	router := httphandler.NewRouter(handler)
	mux := router.SetupRoutes()

	// Настройка сервера
	srv := &http.Server{
		Addr:         ":" + *port,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		log.Printf("Server starting on port %s", *port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Ожидание сигнала для graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}
