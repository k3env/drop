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

	mr, err := c.Request().MultipartReader()
	if err != nil {
		c.Logger().Error("MultipartReader error", "error", err)
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Failed to read request",
		})
	}

	var filename string
	var fileReader io.Reader

	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "invalid_request",
				Message: "Failed to read form data",
			})
		}

		if part.FormName() == "file" {
			filename = part.FileName()
			fileReader = part
			break
		}
	}

	if filename == "" || fileReader == nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "No file provided",
		})
	}

	c.Logger().Info("file received", "filename", filename, "size", contentLength)

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

	result, err := h.service.Upload(c.Request().Context(), fileReader, filename, ttl, oneTimePtr)
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
		ExpiresAt:   *file.ExpiresAt,
		Status:      string(file.GetStatus()),
	})
}
