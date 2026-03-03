package middleware

import (
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
			logger.Info().
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
	return middleware.Recover()
}

func RateLimit(cfg *config.Config) echo.MiddlewareFunc {
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
	return middleware.CORS()
}

func BodyLimit(size int64) echo.MiddlewareFunc {
	return middleware.BodyLimit(size)
}
