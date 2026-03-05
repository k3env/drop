package main

import (
	"context"
	"fmt"
	"html/template"
	"os"
	"os/signal"
	"syscall"

	"github.com/k3env/drop/config"
	"github.com/k3env/drop/handlers"
	"github.com/k3env/drop/middleware"
	"github.com/k3env/drop/repository"
	"github.com/k3env/drop/service"
	"github.com/k3env/drop/storage"
	kx "github.com/k3env/koanf-extensions"
	"github.com/k3env/otlp-init/otlp"
	"github.com/labstack/echo/v5"
	"github.com/rs/zerolog"
)

func main() {
	ctx := context.Background()
	logger := zerolog.New(os.Stderr).With().Timestamp().Logger()
	var cfg = &config.Config{}

	if err := kx.Load(cfg, "DROP"); err != nil {
		logger.Fatal().Err(err).Msg("failed to load config")
	}

	stopFn := otlp.Init(ctx, cfg.Log, "drop.api")
	defer stopFn()

	httpMetrics, err := otlp.NewHTTPMetrics()
	if err != nil {
		logger.Fatal().Err(err).Msg("Error initializing metrics")
	}
	httpTracing := otlp.NewHTTPTracing()

	cw := zerolog.ConsoleWriter{Out: os.Stdout}
	ow := otlp.NewLogger(ctx)

	writer := zerolog.MultiLevelWriter(cw, ow)

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	logger = zerolog.New(writer).With().Timestamp().Logger()
	repo, err := repository.NewRepository(cfg)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to create repository")
	}

	if err := repo.Init(context.Background()); err != nil {
		logger.Fatal().Err(err).Msg("failed to init repository")
	}
	defer repo.Close(context.Background())

	stor, err := storage.NewLocalStorage(cfg)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to create storage")
	}

	if err := stor.Init(context.Background()); err != nil {
		logger.Fatal().Err(err).Msg("failed to init storage")
	}
	defer stor.Close(context.Background())

	fileSvc := service.NewFileService(repo, stor, cfg)

	e := echo.New()

	tmpl := template.Must(template.ParseGlob("ui/templates/*.html"))
	e.Renderer = &echo.TemplateRenderer{
		Template: tmpl,
	}

	e.Use(httpMetrics.Middleware())
	e.Use(httpTracing.Middleware())
	e.Use(middleware.Logger(logger))
	e.Use(middleware.Recovery())
	e.Use(middleware.RateLimit(cfg))
	e.Use(middleware.CORS())
	e.Use(middleware.BodyLimit(cfg.Storage.MaxFileSizeMB * 1024 * 1024))

	e.GET("/health", handlers.Health)
	e.GET("/ready", handlers.Ready)

	fileHandler := handlers.NewFileHandler(fileSvc, cfg)

	e.POST("/api/v1/files", fileHandler.Upload)
	e.GET("/api/v1/files/:token", fileHandler.Metadata)
	e.GET("/:token/download", fileHandler.Download)
	e.DELETE("/api/v1/files/:token", fileHandler.Delete)

	e.Static("/static", "ui/static")
	e.GET("/", handlers.Index)
	e.GET("/:token", handlers.DownloadPage)

	go func() {
		addr := fmt.Sprintf(":%d", cfg.App.Port)
		logger.Info().Str("addr", addr).Msg("starting server")
		if err := e.Start(addr); err != nil {
			logger.Info().Msg("server stopped")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("Server stopped")
}
