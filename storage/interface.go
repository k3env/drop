package storage

import (
	"context"
	"io"
)

type FileStorage interface {
	Save(ctx context.Context, token string, reader io.Reader, filename string) (string, int64, error)
	Load(ctx context.Context, path string) (io.ReadCloser, error)
	Delete(ctx context.Context, path string) error
	Init(ctx context.Context) error
	Close(ctx context.Context) error
	TempDir() string
}
