package repository

import (
	"errors"

	"github.com/k3env/drop/config"
)

func NewRepository(cfg *config.Config) (FileRepository, error) {
	switch cfg.Database.Type {
	case "postgres":
		if cfg.Database.DSN == "" {
			return nil, errors.New("DB_DSN is required for postgres")
		}
		return NewPostgresRepo(cfg.Database.DSN)
	case "duckdb":
		dsn := cfg.Database.DSN
		if dsn == "" {
			dsn = "./meta.db"
		}
		return NewDuckDBRepo(dsn)
	default:
		return nil, errors.New("unsupported DB_TYPE: use postgres or duckdb")
	}
}
