package models

import "time"

type UploadRequest struct {
	File    string `json:"file" form:"file"`
	TTL     *int64 `json:"ttl_seconds" form:"ttl_seconds"`
	OneTime *bool  `json:"one_time" form:"one_time"`
}

type UploadResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

type DeleteResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

type FileMetadataResponse struct {
	Token       string    `json:"token"`
	Filename    string    `json:"filename"`
	Size        int64     `json:"size"`
	ContentType string    `json:"content_type"`
	OneTime     bool      `json:"one_time"`
	ExpiresAt   time.Time `json:"expires_at"`
	Status      string    `json:"status"`
}
