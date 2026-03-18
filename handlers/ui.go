package handlers

import (
	"errors"
	"io/fs"
	"mime"
	"net/http"
	"path/filepath"

	"github.com/labstack/echo/v5"
)

type UIHandler struct {
	uifs fs.FS
}

func NewUIHandler(uiFS fs.FS) *UIHandler {
	return &UIHandler{uifs: uiFS}
}

func (h *UIHandler) ServeSPA(c *echo.Context) error {
	path := c.Param("*")
	if path == "" || path == "/" {
		c.Response().Header().Add("Content-Type", "text/html; charset=utf-8")
		return c.FileFS("index.html", h.uifs)
	}
	if _, err := h.uifs.Open(path); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			c.Response().Header().Add("Content-Type", "text/html; charset=utf-8")
			return c.FileFS("index.html", h.uifs)
		}
		return c.JSON(http.StatusInternalServerError, err)
	}
	ext := filepath.Ext(path)
	c.Response().Header().Add("Content-Type", mime.TypeByExtension(ext))
	return c.FileFS(path, h.uifs)
}
