package models

import (
	"time"

	"github.com/google/uuid"
)

type File struct {
	ID          uuid.UUID  `json:"id"`
	Token       string     `json:"token"`
	Filename    string     `json:"filename"`
	ContentType string     `json:"content_type"`
	Size        int64      `json:"size"`
	StoragePath string     `json:"-"`
	OneTime     bool       `json:"one_time"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	Downloaded  bool       `json:"downloaded"`
	CreatedAt   time.Time  `json:"created_at"`
}

type FileStatus string

const (
	StatusActive   FileStatus = "active"
	StatusExpired  FileStatus = "expired"
	StatusDeleted  FileStatus = "deleted"
	StatusConsumed FileStatus = "consumed"
)

func (f *File) GetStatus() FileStatus {
	if f.Downloaded && f.OneTime {
		return StatusConsumed
	}
	if f.ExpiresAt != nil && time.Now().After(*f.ExpiresAt) {
		return StatusExpired
	}
	return StatusActive
}

func (f *File) IsExpired() bool {
	if f.ExpiresAt != nil && time.Now().After(*f.ExpiresAt) {
		return true
	}
	return false
}
