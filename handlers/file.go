package handlers

import (
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/k3env/drop/config"
	"github.com/k3env/drop/models"
	"github.com/k3env/drop/service"
	"github.com/labstack/echo/v5"
)

type FileHandler struct {
	service *service.FileService
	cfg     *config.Config
}

func NewFileHandler(svc *service.FileService, cfg *config.Config) *FileHandler {
	return &FileHandler{
		service: svc,
		cfg:     cfg,
	}
}

func (h *FileHandler) Upload(c *echo.Context) error {
	file, err := c.FormFile("file")
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "No file provided",
		})
	}

	if file.Size > h.cfg.Storage.MaxFileSizeMB {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "file_too_large",
			Message: "File exceeds maximum allowed size",
		})
	}

	src, err := file.Open()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "upload_error",
			Message: "Failed to open file",
		})
	}
	defer src.Close()

	var ttl *int64
	if ttlStr := c.FormValue("ttl_seconds"); ttlStr != "" {
		if ttlVal, err := strconv.ParseInt(ttlStr, 10, 64); err == nil {
			ttl = &ttlVal
		}
	}

	oneTime := false
	if ot := c.FormValue("one_time"); ot != "" {
		oneTime = ot == "true" || ot == "1"
	}
	oneTimePtr := &oneTime

	result, err := h.service.Upload(c.Request().Context(), src, file.Filename, ttl, oneTimePtr)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "upload_failed",
			Message: err.Error(),
		})
	}

	return c.JSON(http.StatusCreated, models.UploadResponse{
		Token:     result.Token,
		ExpiresAt: *result.ExpiresAt,
	})
}

func (h *FileHandler) Download(c *echo.Context) error {
	token := c.Param("token")
	if token == "" {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Token is required",
		})
	}

	reader, file, err := h.service.Download(c.Request().Context(), token)
	if err != nil {
		if errors.Is(err, service.ErrFileNotFound) {
			return c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "file_not_found",
				Message: "File does not exist",
			})
		}
		if errors.Is(err, service.ErrFileExpired) {
			return c.JSON(http.StatusGone, models.ErrorResponse{
				Error:   "file_expired",
				Message: "File has expired",
			})
		}
		if errors.Is(err, service.ErrFileConsumed) {
			return c.JSON(http.StatusForbidden, models.ErrorResponse{
				Error:   "file_already_downloaded",
				Message: "File can only be downloaded once",
			})
		}
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "download_failed",
			Message: "Failed to download file",
		})
	}
	defer reader.Close()

	c.Response().Header().Set("Content-Disposition", "attachment; filename="+file.Filename)
	c.Response().Header().Set("Content-Type", file.ContentType)
	_, err = io.Copy(c.Response(), reader)
	return err
}

func (h *FileHandler) Delete(c *echo.Context) error {
	token := c.Param("token")
	if token == "" {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Token is required",
		})
	}

	err := h.service.Delete(c.Request().Context(), token)
	if err != nil {
		if errors.Is(err, service.ErrFileNotFound) {
			return c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "file_not_found",
				Message: "File does not exist",
			})
		}
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "delete_failed",
			Message: "Failed to delete file",
		})
	}

	return c.JSON(http.StatusOK, models.DeleteResponse{
		Success: true,
		Message: "File deleted successfully",
	})
}
