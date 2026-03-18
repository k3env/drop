package middleware

import (
	"fmt"
	"net/http"

	"github.com/k3env/drop/config"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"github.com/rs/zerolog"
	"golang.org/x/time/rate"
)

func Logger(logger zerolog.Logger) echo.MiddlewareFunc {
	mw := middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogStatus:  true,
		LogMethod:  true,
		LogURI:     true,
		LogLatency: true,
		LogValuesFunc: func(c *echo.Context, v middleware.RequestLoggerValues) error {
			event := logger.Info()
			if v.Status >= 500 {
				event = logger.Error()
			} else if v.Status >= 400 {
				event = logger.Warn()
			}
			event.
				Str("method", c.Request().Method).
				Str("path", c.Request().URL.Path).
				Int("status", v.Status).
				Dur("latency", v.Latency).
				Str("ip", c.RealIP()).
				Msg("request")
			return nil
		},
	})
	return mw
}

func Recovery() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			defer func() {
				if r := recover(); r != nil {
					err, ok := r.(error)
					if !ok {
						err = fmt.Errorf("%v", r)
					}
					logger := zerolog.Ctx(c.Request().Context())
					logger.Error().
						Err(err).
						Str("method", c.Request().Method).
						Str("path", c.Request().URL.Path).
						Str("ip", c.RealIP()).
						Msg("panic recovered")
					c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal_error"})
				}
			}()
			return next(c)
		}
	}
}

func RateLimit(cfg *config.Config) echo.MiddlewareFunc {
	if !cfg.RateLimit.Enabled {
		return func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c *echo.Context) error {
				return next(c)
			}
		}
	}

	limiter := rate.NewLimiter(
		rate.Limit(float64(cfg.RateLimit.RPM)/60),
		cfg.RateLimit.Burst,
	)

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			if !limiter.Allow() {
				return echo.NewHTTPError(429, "rate limit exceeded")
			}
			return next(c)
		}
	}
}

func CORS() echo.MiddlewareFunc {
	return middleware.CORSWithConfig(
		middleware.CORSConfig{
			AllowOrigins: []string{"*"},
			UnsafeAllowOriginFunc: func(c *echo.Context, origin string) (allowedOrigin string, allowed bool, err error) {
				return c.Request().Header.Get("Origin"), true, nil
			},
		},
	)
}

func BodyLimit(size int64) echo.MiddlewareFunc {
	return middleware.BodyLimit(size)
}
