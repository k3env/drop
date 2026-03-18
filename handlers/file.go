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
	contentLength := c.Request().ContentLength
	if contentLength > h.cfg.Storage.MaxFileSizeMB*1024*1024 {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "file_too_large",
			Message: "File exceeds maximum allowed size",
		})
	}
	var request models.UploadRequest
	if err := c.Bind(&request); err != nil {
		c.Logger().Error("Request bind error", "error", err)
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "bad_request",
			Message: "Failed to bind request",
		})
	}

	filename := request.File.Filename
	ttlSeconds := request.TTL

	c.Logger().Info("file received", "filename", filename, "ttl", ttlSeconds, "one_time", request.OneTime)
	f, err := request.File.Open()
	if err != nil {
		c.Logger().Error("Failed to open file", "error", err)
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "bad_request",
			Message: "Failed to open file",
		})
	}

	result, err := h.service.Upload(c.Request().Context(), f, filename, ttlSeconds, request.OneTime)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "upload_failed",
			Message: err.Error(),
		})
	}

	return c.JSON(http.StatusCreated, models.UploadResponse{
		Token:     result.Token,
		ExpiresAt: result.ExpiresAt,
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
	c.Response().Header().Set("Content-Length", strconv.FormatInt(file.Size, 10))
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

func (h *FileHandler) Metadata(c *echo.Context) error {
	token := c.Param("token")
	if token == "" {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Token is required",
		})
	}

	file, err := h.service.GetMetadata(c.Request().Context(), token)
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
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "metadata_failed",
			Message: "Failed to get file metadata",
		})
	}

	return c.JSON(http.StatusOK, models.FileMetadataResponse{
		Token:       file.Token,
		Filename:    file.Filename,
		Size:        file.Size,
		ContentType: file.ContentType,
		OneTime:     file.OneTime,
		ExpiresAt:   file.ExpiresAt,
		Status:      string(file.GetStatus()),
	})
}
