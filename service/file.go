package service

import (
	"context"
	"crypto/rand"
	"errors"
	"io"
	"mime"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/k3env/drop/config"
	"github.com/k3env/drop/models"
	"github.com/k3env/drop/repository"
	"github.com/k3env/drop/storage"
)

var (
	ErrFileNotFound   = errors.New("file_not_found")
	ErrFileExpired    = errors.New("file_expired")
	ErrFileConsumed   = errors.New("file_already_downloaded")
	ErrInvalidToken   = errors.New("invalid_token")
	ErrTokenCollision = errors.New("token_collision")
)

const (
	tokenCharset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	minTokenLen  = 6
	maxTokenLen  = 12
	maxRetries   = 5
)

type FileService struct {
	repo    repository.FileRepository
	storage storage.FileStorage
	cfg     *config.Config
}

func NewFileService(repo repository.FileRepository, storage storage.FileStorage, cfg *config.Config) *FileService {
	return &FileService{
		repo:    repo,
		storage: storage,
		cfg:     cfg,
	}
}

func (s *FileService) Upload(ctx context.Context, reader io.Reader, filename string, ttl *int64, oneTime *bool) (*models.File, error) {
	token, err := s.generateToken()
	if err != nil {
		return nil, err
	}

	contentType := mime.TypeByExtension(filepath.Ext(filename))
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	storagePath, err := s.storage.Save(ctx, token, reader, filename)
	if err != nil {
		return nil, err
	}

	expiresAt := time.Now().Add(s.cfg.Storage.DefaultTTL)
	if ttl != nil && *ttl > 0 {
		expiresAt = time.Now().Add(time.Duration(*ttl) * time.Second)
	}

	useOneTime := false
	if oneTime != nil {
		useOneTime = *oneTime
	}

	file := &models.File{
		ID:          uuid.New(),
		Token:       token,
		Filename:    filepath.Base(filename),
		ContentType: contentType,
		Size:        0,
		StoragePath: storagePath,
		OneTime:     useOneTime,
		ExpiresAt:   &expiresAt,
		Downloaded:  false,
		CreatedAt:   time.Now(),
	}

	if err := s.repo.Create(ctx, file); err != nil {
		s.storage.Delete(ctx, storagePath)
		return nil, err
	}

	return file, nil
}

func (s *FileService) Download(ctx context.Context, token string) (io.ReadCloser, *models.File, error) {
	file, err := s.repo.GetByToken(ctx, token)
	if err != nil {
		return nil, nil, err
	}
	if file == nil {
		return nil, nil, ErrFileNotFound
	}

	if file.IsExpired() {
		return nil, nil, ErrFileExpired
	}

	if file.OneTime && file.Downloaded {
		return nil, nil, ErrFileConsumed
	}

	reader, err := s.storage.Load(ctx, file.StoragePath)
	if err != nil {
		return nil, nil, err
	}

	if file.OneTime {
		if err := s.repo.MarkDownloaded(ctx, token); err != nil {
			reader.Close()
			return nil, nil, err
		}
		file.Downloaded = true
	}

	return reader, file, nil
}

func (s *FileService) Delete(ctx context.Context, token string) error {
	file, err := s.repo.GetByToken(ctx, token)
	if err != nil {
		return err
	}
	if file == nil {
		return ErrFileNotFound
	}

	if err := s.storage.Delete(ctx, file.StoragePath); err != nil {
		return err
	}

	return s.repo.Delete(ctx, token)
}

func (s *FileService) generateToken() (string, error) {
	for i := 0; i < maxRetries; i++ {
		length := minTokenLen + i*2
		if length > maxTokenLen {
			length = maxTokenLen
		}

		token, err := generateRandomString(length)
		if err != nil {
			continue
		}

		exists, err := s.tokenExists(context.Background(), token)
		if err != nil {
			continue
		}
		if !exists {
			return token, nil
		}
	}
	return "", ErrTokenCollision
}

func (s *FileService) tokenExists(ctx context.Context, token string) (bool, error) {
	f, err := s.repo.GetByToken(ctx, token)
	if err != nil {
		return false, err
	}
	return f != nil, nil
}

func generateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	result := make([]byte, length)
	for i := 0; i < length; i++ {
		result[i] = tokenCharset[int(bytes[i])%len(tokenCharset)]
	}
	return string(result), nil
}
