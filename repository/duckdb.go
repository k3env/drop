package repository

import (
	"context"
	"errors"

	"github.com/k3env/drop/models"
)

func NewDuckDBRepo(dsn string) (*DuckDBRepo, error) {
	return nil, errors.New("duckdb support requires building with -tags duckdb and installing github.com/marcboeker/go-duckdb")
}

type DuckDBRepo struct{}

func (r *DuckDBRepo) Init(ctx context.Context) error {
	return errors.New("duckdb not supported")
}

func (r *DuckDBRepo) Close(ctx context.Context) error {
	return nil
}

func (r *DuckDBRepo) Create(ctx context.Context, file *models.File) error {
	return errors.New("duckdb not supported")
}

func (r *DuckDBRepo) GetByToken(ctx context.Context, token string) (*models.File, error) {
	return nil, errors.New("duckdb not supported")
}

func (r *DuckDBRepo) GetByID(ctx context.Context, id string) (*models.File, error) {
	return nil, errors.New("duckdb not supported")
}

func (r *DuckDBRepo) Update(ctx context.Context, file *models.File) error {
	return errors.New("duckdb not supported")
}

func (r *DuckDBRepo) Delete(ctx context.Context, token string) error {
	return errors.New("duckdb not supported")
}

func (r *DuckDBRepo) MarkDownloaded(ctx context.Context, token string) error {
	return errors.New("duckdb not supported")
}
