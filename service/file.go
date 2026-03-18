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
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("drop.service")

func WithLogger(ctx context.Context, logger zerolog.Logger) context.Context {
	return logger.WithContext(ctx)
}

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

func (s *FileService) TempDir() string {
	return s.storage.TempDir()
}

func (s *FileService) Upload(ctx context.Context, reader io.Reader, filename string, ttl *int64, oneTime *bool) (*models.File, error) {
	ctx, span := tracer.Start(ctx, "FileService.Upload",
		trace.WithAttributes(attribute.String("filename", filename)))
	defer span.End()
	var logger zerolog.Logger

	token, err := s.generateToken()
	if err != nil {
		zerolog.Ctx(ctx).Error().Err(err).Msg("failed to generate token")
		span.RecordError(err)
		return nil, err
	}
	logger = zerolog.Ctx(ctx).With().Str("token", token).Logger()
	ctx = logger.WithContext(ctx)
	span.SetAttributes(attribute.String("token", token))

	contentType := mime.TypeByExtension(filepath.Ext(filename))
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	ctxStorage, spanStorage := tracer.Start(ctx, "storage.Save")
	storagePath, size, err := s.storage.Save(ctxStorage, token, reader, filename)
	spanStorage.End()
	if err != nil {
		logger.Error().Err(err).Str("storage_path", storagePath).Msg("failed to save file")
		span.RecordError(err)
		return nil, err
	}

	expiresAt := time.Now().UTC().Add(s.cfg.Storage.DefaultTTL)
	if ttl != nil && *ttl > 0 {
		expiresAt = time.Now().UTC().Add(time.Duration(*ttl) * time.Second)
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
		Size:        size,
		StoragePath: storagePath,
		OneTime:     useOneTime,
		ExpiresAt:   &expiresAt,
		Downloaded:  false,
		CreatedAt:   time.Now().UTC(),
	}
	if ttl == nil || *ttl == 0 {
		file.ExpiresAt = nil
	}

	ctxRepo, spanRepo := tracer.Start(ctx, "repository.Create")
	if err := s.repo.Create(ctxRepo, file); err != nil {
		logger.Error().Err(err).Msg("failed to create file record")
		spanRepo.RecordError(err)
		spanRepo.End()
		s.storage.Delete(ctx, storagePath)
		span.RecordError(err)
		return nil, err
	}
	spanRepo.End()

	logger.Info().
		Str("file_id", file.ID.String()).
		Int64("size", size).
		Bool("one_time", useOneTime).
		Msg("file uploaded")

	span.SetAttributes(
		attribute.String("file.id", file.ID.String()),
		attribute.Bool("one_time", useOneTime),
	)

	return file, nil
}

func (s *FileService) Download(ctx context.Context, token string) (io.ReadCloser, *models.File, error) {
	ctx, span := tracer.Start(ctx, "FileService.Download",
		trace.WithAttributes(attribute.String("token", token)))
	defer span.End()
	log := zerolog.Ctx(ctx)

	ctxRepo, spanRepo := tracer.Start(ctx, "repository.GetByToken")
	file, err := s.repo.GetByToken(ctxRepo, token)
	spanRepo.End()

	if err != nil {
		log.Error().Err(err).Str("token", token).Msg("failed to get file by token")
		span.RecordError(err)
		return nil, nil, err
	}
	if file == nil {
		log.Warn().Str("token", token).Msg("file not found")
		span.SetAttributes(attribute.String("error", "file_not_found"))
		return nil, nil, ErrFileNotFound
	}

	if file.IsExpired() {
		log.Info().Str("token", token).Str("file_id", file.ID.String()).Msg("file expired")
		span.SetAttributes(attribute.String("error", "file_expired"))
		return nil, nil, ErrFileExpired
	}

	if file.OneTime && file.Downloaded {
		log.Info().Str("token", token).Str("file_id", file.ID.String()).Msg("file already downloaded")
		span.SetAttributes(attribute.String("error", "file_consumed"))
		return nil, nil, ErrFileConsumed
	}

	span.SetAttributes(
		attribute.String("file.id", file.ID.String()),
		attribute.String("filename", file.Filename),
		attribute.Bool("one_time", file.OneTime),
		attribute.Bool("file.downloaded", file.Downloaded),
	)

	ctxStorage, spanStorage := tracer.Start(ctx, "storage.Load")
	reader, err := s.storage.Load(ctxStorage, file.StoragePath)
	spanStorage.End()

	if err != nil {
		log.Error().Err(err).Msg("failed to load file from storage")
		span.RecordError(err)
		return nil, nil, err
	}

	if file.OneTime {
		span.SetAttributes(attribute.String("action", "marking_downloaded"))
		ctxRepo2, spanRepo2 := tracer.Start(ctx, "repository.MarkDownloaded")
		if err := s.repo.MarkDownloaded(ctxRepo2, token); err != nil {
			log.Error().Err(err).Msg("failed to mark file as downloaded")
			spanRepo2.RecordError(err)
			spanRepo2.End()
			reader.Close()
			span.RecordError(err)
			return nil, nil, err
		}
		spanRepo2.End()
		file.Downloaded = true
	}

	log.Info().
		Str("file_id", file.ID.String()).
		Str("filename", file.Filename).
		Int64("size", file.Size).
		Msg("file downloaded")

	span.SetAttributes(
		attribute.String("file.id", file.ID.String()),
		attribute.String("filename", file.Filename),
		attribute.Bool("one_time", file.OneTime),
	)

	return reader, file, nil
}

func (s *FileService) Delete(ctx context.Context, token string) error {
	ctx, span := tracer.Start(ctx, "FileService.Delete",
		trace.WithAttributes(attribute.String("token", token)))
	defer span.End()
	log := zerolog.Ctx(ctx)

	ctxRepo, spanRepo := tracer.Start(ctx, "repository.GetByToken")
	file, err := s.repo.GetByToken(ctxRepo, token)
	spanRepo.End()

	if err != nil {
		log.Error().Err(err).Str("token", token).Msg("failed to get file by token")
		span.RecordError(err)
		return err
	}
	if file == nil {
		log.Warn().Str("token", token).Msg("file not found")
		span.SetAttributes(attribute.String("error", "file_not_found"))
		return ErrFileNotFound
	}

	span.SetAttributes(attribute.String("file.id", file.ID.String()))

	ctxStorage, spanStorage := tracer.Start(ctx, "storage.Delete")
	if err := s.storage.Delete(ctxStorage, file.StoragePath); err != nil {
		log.Error().Err(err).Str("storage_path", file.StoragePath).Msg("failed to delete file from storage")
		spanStorage.RecordError(err)
		spanStorage.End()
		span.RecordError(err)
		return err
	}
	spanStorage.End()

	ctxRepo2, spanRepo2 := tracer.Start(ctx, "repository.Delete")
	if err := s.repo.Delete(ctxRepo2, token); err != nil {
		log.Error().Err(err).Msg("failed to delete file record")
		spanRepo2.RecordError(err)
		spanRepo2.End()
		span.RecordError(err)
		return err
	}
	spanRepo2.End()

	log.Info().Str("file_id", file.ID.String()).Msg("file deleted")

	return nil
}

func (s *FileService) GetMetadata(ctx context.Context, token string) (*models.File, error) {
	ctx, span := tracer.Start(ctx, "FileService.GetMetadata",
		trace.WithAttributes(attribute.String("token", token)))
	defer span.End()
	log := zerolog.Ctx(ctx)

	ctxRepo, spanRepo := tracer.Start(ctx, "repository.GetByToken")
	file, err := s.repo.GetByToken(ctxRepo, token)
	spanRepo.End()

	if err != nil {
		log.Error().Err(err).Str("token", token).Msg("failed to get file by token")
		span.RecordError(err)
		return nil, err
	}
	if file == nil {
		log.Warn().Str("token", token).Msg("file not found")
		span.SetAttributes(attribute.String("error", "file_not_found"))
		return nil, ErrFileNotFound
	}

	span.SetAttributes(
		attribute.String("file.id", file.ID.String()),
		attribute.String("filename", file.Filename),
	)

	return file, nil
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
