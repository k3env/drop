package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
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

//go:embed ui
var embeddedFS embed.FS

func spaErrorHandler(uiFS fs.FS) echo.HTTPErrorHandler {
	return func(c *echo.Context, err error) {
		he, ok := err.(*echo.HTTPError)
		if !ok {
			c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal_error"})
			return
		}

		if he.Code != http.StatusNotFound {
			c.JSON(he.Code, map[string]string{"error": fmt.Sprintf("%v", he.Message)})
			return
		}

		path := c.Request().URL.Path
		if path != "/index.html" && !isAssetPath(path) {
			if _, openErr := uiFS.Open("index.html"); openErr == nil {
				c.File("index.html")
				return
			}
		}

		c.JSON(http.StatusNotFound, map[string]string{"error": "not_found"})
	}
}

func isAssetPath(path string) bool {
	ext := filepath.Ext(path)
	return ext == ".js" || ext == ".css" || ext == ".svg" || ext == ".png" || ext == ".jpg" || ext == ".ico"
}

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

	e.Use(httpMetrics.Middleware())
	e.Use(httpTracing.Middleware())
	e.Use(middleware.Logger(logger))
	e.Use(middleware.Recovery())
	e.Use(middleware.RateLimit(cfg))
	e.Use(middleware.CORS())
	e.Use(middleware.BodyLimit(cfg.Storage.MaxFileSizeMB * 1024 * 1024))
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			ctx := logger.WithContext(c.Request().Context())
			c.SetRequest(c.Request().WithContext(ctx))
			return next(c)
		}
	})

	e.GET("/health", handlers.Health)
	e.GET("/ready", handlers.Ready)

	fileHandler := handlers.NewFileHandler(fileSvc, cfg)

	e.POST("/api/v1/files", fileHandler.Upload)
	e.GET("/api/v1/files/:token", fileHandler.Metadata)
	e.GET("/:token/download", fileHandler.Download)
	e.DELETE("/api/v1/files/:token", fileHandler.Delete)

	staticFS, _ := fs.Sub(embeddedFS, "ui")
	spa := handlers.NewUIHandler(staticFS)
	e.GET("/*", spa.ServeSPA)
	//e.HTTPErrorHandler = spaErrorHandler(staticFS)

	go func() {
		addr := fmt.Sprintf(":%d", cfg.App.Port)
		logger.Info().Str("addr", addr).Msg("starting server")
		if err := e.Start(addr); err != nil && err != http.ErrServerClosed {
			logger.Error().Err(err).Msg("server error")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("Server stopped")
}
