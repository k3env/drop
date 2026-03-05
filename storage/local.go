package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/k3env/drop/config"
)

type LocalStorage struct {
	basePath string
}

func NewLocalStorage(cfg *config.Config) (*LocalStorage, error) {
	return &LocalStorage{
		basePath: cfg.Storage.Local.Path,
	}, nil
}

func (s *LocalStorage) Init(ctx context.Context) error {
	return os.MkdirAll(s.basePath, 0755)
}

func (s *LocalStorage) Close(ctx context.Context) error {
	return nil
}

func (s *LocalStorage) Save(ctx context.Context, token string, reader io.Reader, filename string) (string, int64, error) {
	dir := filepath.Join(s.basePath, token[:2])
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", 0, err
	}

	path := filepath.Join(dir, token)
	file, err := os.Create(path)
	if err != nil {
		return "", 0, err
	}
	defer file.Close()

	written, err := io.Copy(file, reader)
	if err != nil {
		os.Remove(path)
		return "", 0, err
	}

	if written == 0 {
		os.Remove(path)
		return "", 0, errors.New("empty file")
	}

	return path, written, nil
}

func (s *LocalStorage) Load(ctx context.Context, path string) (io.ReadCloser, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s", path)
		}
		return nil, err
	}
	return file, nil
}

func (s *LocalStorage) Delete(ctx context.Context, path string) error {
	return os.Remove(path)
}
