package handlers

import (
	"github.com/labstack/echo/v5"
)

func Index(c *echo.Context) error {
	return c.Render(200, "index.html", nil)
}

type DownloadPageData struct {
	Token string
	URL   string
}

func DownloadPage(c *echo.Context) error {
	token := c.Param("token")
	if token == "" {
		return c.Redirect(302, "/")
	}

	scheme := "http"
	if c.Request().TLS != nil || c.Request().Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	host := c.Request().Host
	url := scheme + "://" + host + "/" + token

	return c.Render(200, "download.html", DownloadPageData{
		Token: token,
		URL:   url,
	})
}
