package service

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/k3env/drop/config"
	"github.com/k3env/drop/models"
	"github.com/k3env/drop/service/mocks"
)

func testConfig() *config.Config {
	return &config.Config{
		App: &config.AppConfig{
			Port: 8080,
		},
		Storage: &config.StorageConfig{
			MaxFileSizeMB: 100,
			DefaultTTL:    24 * time.Hour,
			Type:          "local",
			Local: &config.LocalStorageConfig{
				Path: "./testdata",
			},
		},
		Database: &config.DatabaseConfig{
			Type: "duckdb",
			DSN:  "./meta.test.db",
		},
		RateLimit: &config.RateLimitConfig{
			Enabled: false,
			RPM:     60,
			Burst:   10,
		},
	}
}

func TestGenerateToken(t *testing.T) {
	repo := mocks.NewMockFileRepository()
	stor := mocks.NewMockFileStorage()
	cfg := testConfig()

	svc := NewFileService(repo, stor, cfg)

	token, err := svc.generateToken()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(token) < 6 || len(token) > 12 {
		t.Errorf("expected token length between 6 and 12, got %d", len(token))
	}
}

func TestUpload(t *testing.T) {
	repo := mocks.NewMockFileRepository()
	stor := mocks.NewMockFileStorage()
	cfg := testConfig()

	svc := NewFileService(repo, stor, cfg)

	reader := bytes.NewReader([]byte("test file content"))
	ttl := int64(3600)
	oneTime := true

	file, err := svc.Upload(context.Background(), reader, "test.txt", &ttl, &oneTime)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if file.Token == "" {
		t.Error("expected token to be generated")
	}

	if file.Filename != "test.txt" {
		t.Errorf("expected filename test.txt, got %s", file.Filename)
	}

	if !file.OneTime {
		t.Error("expected one_time to be true")
	}

	if file.ExpiresAt == nil {
		t.Error("expected expires_at to be set")
	}
}

func TestUploadWithDefaultTTL(t *testing.T) {
	repo := mocks.NewMockFileRepository()
	stor := mocks.NewMockFileStorage()
	cfg := testConfig()
	cfg.Storage.DefaultTTL = 7200 * time.Second

	svc := NewFileService(repo, stor, cfg)

	reader := bytes.NewReader([]byte("test"))

	file, err := svc.Upload(context.Background(), reader, "test.txt", nil, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expectedExpiry := time.Now().Add(7200 * time.Second)
	if file.ExpiresAt.Before(expectedExpiry.Add(-time.Second)) || file.ExpiresAt.After(expectedExpiry.Add(time.Second)) {
		t.Error("expected expires_at to be 2 hours from now")
	}
}

func TestDownloadNonExistent(t *testing.T) {
	repo := mocks.NewMockFileRepository()
	stor := mocks.NewMockFileStorage()
	cfg := testConfig()

	svc := NewFileService(repo, stor, cfg)

	_, _, err := svc.Download(context.Background(), "nonexistent")
	if err != ErrFileNotFound {
		t.Errorf("expected ErrFileNotFound, got %v", err)
	}
}

func TestDeleteNonExistent(t *testing.T) {
	repo := mocks.NewMockFileRepository()
	stor := mocks.NewMockFileStorage()
	cfg := testConfig()

	svc := NewFileService(repo, stor, cfg)

	err := svc.Delete(context.Background(), "nonexistent")
	if err != ErrFileNotFound {
		t.Errorf("expected ErrFileNotFound, got %v", err)
	}
}

func TestTokenCollision(t *testing.T) {
	repo := mocks.NewMockFileRepository()
	stor := mocks.NewMockFileStorage()
	cfg := testConfig()

	svc := NewFileService(repo, stor, cfg)

	for i := 0; i < 5; i++ {
		reader := bytes.NewReader([]byte("test"))
		_, err := svc.Upload(context.Background(), reader, "test.txt", nil, nil)
		if err != nil {
			t.Fatalf("expected no error on upload %d, got %v", i, err)
		}
	}

	t.Log("Successfully created 5 files without collision")
}

func TestFileStatus(t *testing.T) {
	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	activeFile := &models.File{
		ExpiresAt:  &future,
		Downloaded: false,
	}
	if activeFile.GetStatus() != models.StatusActive {
		t.Error("expected status active")
	}

	expiredFile := &models.File{
		ExpiresAt:  &past,
		Downloaded: false,
	}
	if expiredFile.GetStatus() != models.StatusExpired {
		t.Error("expected status expired")
	}

	consumedFile := &models.File{
		ExpiresAt:  &future,
		OneTime:    true,
		Downloaded: true,
	}
	if consumedFile.GetStatus() != models.StatusConsumed {
		t.Error("expected status consumed")
	}
}

func TestIsExpired(t *testing.T) {
	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	expiredFile := &models.File{
		ExpiresAt: &past,
	}
	if !expiredFile.IsExpired() {
		t.Error("expected file to be expired")
	}

	activeFile := &models.File{
		ExpiresAt: &future,
	}
	if activeFile.IsExpired() {
		t.Error("expected file to not be expired")
	}

	noExpiry := &models.File{}
	if noExpiry.IsExpired() {
		t.Error("expected file without expiry to not be expired")
	}
}
