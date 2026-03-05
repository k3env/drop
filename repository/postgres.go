package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/k3env/drop/models"
)

type PostgresRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresRepo(dsn string) (*PostgresRepo, error) {
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		return nil, err
	}
	return &PostgresRepo{pool: pool}, nil
}

func (r *PostgresRepo) Init(ctx context.Context) error {
	query := `
	CREATE TABLE IF NOT EXISTS files (
		id UUID PRIMARY KEY,
		token VARCHAR(12) UNIQUE NOT NULL,
		filename VARCHAR(255) NOT NULL,
		content_type VARCHAR(100) NOT NULL,
		size BIGINT NOT NULL,
		storage_path VARCHAR(512) NOT NULL,
		one_time BOOLEAN DEFAULT FALSE,
		expires_at TIMESTAMP,
		downloaded BOOLEAN DEFAULT FALSE,
		created_at TIMESTAMP DEFAULT NOW()
	);
	CREATE INDEX IF NOT EXISTS idx_files_token ON files(token);
	`
	_, err := r.pool.Exec(ctx, query)
	return err
}

func (r *PostgresRepo) Close(ctx context.Context) error {
	r.pool.Close()
	return nil
}

func (r *PostgresRepo) Create(ctx context.Context, file *models.File) error {
	query := `
	INSERT INTO files (id, token, filename, content_type, size, storage_path, one_time, expires_at, downloaded, created_at)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := r.pool.Exec(ctx, query,
		file.ID, file.Token, file.Filename, file.ContentType,
		file.Size, file.StoragePath, file.OneTime, file.ExpiresAt,
		file.Downloaded, file.CreatedAt)
	return err
}

func (r *PostgresRepo) GetByToken(ctx context.Context, token string) (*models.File, error) {
	query := `
	SELECT id, token, filename, content_type, size, storage_path, one_time, expires_at, downloaded, created_at
	FROM files WHERE token = $1
	`
	row := r.pool.QueryRow(ctx, query, token)
	return scanPostgresFile(row)
}

func (r *PostgresRepo) GetByID(ctx context.Context, id string) (*models.File, error) {
	query := `
	SELECT id, token, filename, content_type, size, storage_path, one_time, expires_at, downloaded, created_at
	FROM files WHERE id = $1
	`
	row := r.pool.QueryRow(ctx, query, id)
	return scanPostgresFile(row)
}

func (r *PostgresRepo) Update(ctx context.Context, file *models.File) error {
	query := `
	UPDATE files SET filename = $2, content_type = $3, size = $4, storage_path = $5,
	one_time = $6, expires_at = $7, downloaded = $8 WHERE token = $1
	`
	_, err := r.pool.Exec(ctx, query,
		file.Token, file.Filename, file.ContentType, file.Size,
		file.StoragePath, file.OneTime, file.ExpiresAt, file.Downloaded)
	return err
}

func (r *PostgresRepo) Delete(ctx context.Context, token string) error {
	query := `DELETE FROM files WHERE token = $1`
	_, err := r.pool.Exec(ctx, query, token)
	return err
}

func (r *PostgresRepo) MarkDownloaded(ctx context.Context, token string) error {
	query := `UPDATE files SET downloaded = TRUE WHERE token = $1`
	_, err := r.pool.Exec(ctx, query, token)
	return err
}

func scanPostgresFile(row pgx.Row) (*models.File, error) {
	var f models.File
	var id string
	err := row.Scan(&id, &f.Token, &f.Filename, &f.ContentType, &f.Size,
		&f.StoragePath, &f.OneTime, &f.ExpiresAt, &f.Downloaded, &f.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	f.ID, _ = uuid.Parse(id)
	return &f, nil
}
