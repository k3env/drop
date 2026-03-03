package repository

import (
	"context"

	"github.com/k3env/drop/models"
)

type FileRepository interface {
	Create(ctx context.Context, file *models.File) error
	GetByToken(ctx context.Context, token string) (*models.File, error)
	GetByID(ctx context.Context, id string) (*models.File, error)
	Update(ctx context.Context, file *models.File) error
	Delete(ctx context.Context, token string) error
	MarkDownloaded(ctx context.Context, token string) error
	Init(ctx context.Context) error
	Close(ctx context.Context) error
}
