package mocks

import (
	"context"
	"io"

	"github.com/k3env/drop/models"
	"github.com/k3env/drop/repository"
	"github.com/k3env/drop/storage"
)

type MockFileRepository struct {
	files map[string]*models.File
}

func NewMockFileRepository() *MockFileRepository {
	return &MockFileRepository{
		files: make(map[string]*models.File),
	}
}

func (r *MockFileRepository) Create(ctx context.Context, file *models.File) error {
	r.files[file.Token] = file
	return nil
}

func (r *MockFileRepository) GetByToken(ctx context.Context, token string) (*models.File, error) {
	return r.files[token], nil
}

func (r *MockFileRepository) GetByID(ctx context.Context, id string) (*models.File, error) {
	for _, f := range r.files {
		if f.ID.String() == id {
			return f, nil
		}
	}
	return nil, nil
}

func (r *MockFileRepository) Update(ctx context.Context, file *models.File) error {
	r.files[file.Token] = file
	return nil
}

func (r *MockFileRepository) Delete(ctx context.Context, token string) error {
	delete(r.files, token)
	return nil
}

func (r *MockFileRepository) MarkDownloaded(ctx context.Context, token string) error {
	if f, ok := r.files[token]; ok {
		f.Downloaded = true
	}
	return nil
}

func (r *MockFileRepository) Init(ctx context.Context) error {
	return nil
}

func (r *MockFileRepository) Close(ctx context.Context) error {
	return nil
}

type MockFileStorage struct {
	files map[string]string
}

func NewMockFileStorage() *MockFileStorage {
	return &MockFileStorage{
		files: make(map[string]string),
	}
}

func (s *MockFileStorage) Save(ctx context.Context, token string, reader io.Reader, filename string) (string, int64, error) {
	path := "/mock/" + token
	s.files[path] = filename
	return path, 0, nil
}

func (s *MockFileStorage) Load(ctx context.Context, path string) (io.ReadCloser, error) {
	return nil, nil
}

func (s *MockFileStorage) Delete(ctx context.Context, path string) error {
	delete(s.files, path)
	return nil
}

func (s *MockFileStorage) Init(ctx context.Context) error {
	return nil
}

func (s *MockFileStorage) Close(ctx context.Context) error {
	return nil
}

var _ repository.FileRepository = (*MockFileRepository)(nil)
var _ storage.FileStorage = (*MockFileStorage)(nil)
