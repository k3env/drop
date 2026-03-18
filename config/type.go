package config

import (
	"time"

	otlp "github.com/k3env/otlp-init/config"
)

type Config struct {
	App       *AppConfig       `config:"app"`
	Storage   *StorageConfig   `config:"storage"`
	Database  *DatabaseConfig  `config:"database"`
	RateLimit *RateLimitConfig `config:"rate_limit"`
	Log       *otlp.LogConfig  `config:"log"`
}

type AppConfig struct {
	Port int `config:"port" usage:"API listen port" default:"8080"`
}

type StorageConfig struct {
	MaxFileSizeMB int64                `config:"max_file_size" usage:"Max file size in MB" default:"100"`
	DefaultTTL    time.Duration        `config:"default_ttl" usage:"Default TTL" default:"24h"`
	Type          string               `config:"type" usage:"Storage type" default:"local"`
	Local         *LocalStorageConfig  `config:"local"`  // use if type is local
	Object        *ObjectStorageConfig `config:"object"` // use if type is object
}

type LocalStorageConfig struct {
	Path    string `config:"path" default:"./data" usage:"Local storage path"`
	TempDir string `config:"temp_dir" default:"" usage:"Temp directory for uploads (empty = system default)"`
}

type ObjectStorageConfig struct{} // not implemented now

type DatabaseConfig struct {
	Type string `config:"type" default:"duckdb" usage:"Database type"`
	DSN  string `config:"dsn" usage:"Database DSN"`
}

type RateLimitConfig struct {
	RPM     int  `config:"rpm" usage:"RPM rate limit" default:"60"`
	Burst   int  `config:"burst" usage:"Burst rate limit" default:"10"`
	Enabled bool `config:"enabled" usage:"Enable rate limit" default:"false"`
}
