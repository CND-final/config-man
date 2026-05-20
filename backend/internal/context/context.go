package context

import (
	stdctx "context"
	"database/sql"
	"fmt"

	"config-man/backend/internal/logger"
	"config-man/backend/internal/store"
	"config-man/backend/pkg/config"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type ConfigManContext struct {
	Config config.Config
	DB     *sql.DB
	Store  *store.Store
}

func NewConfigManContext(ctx stdctx.Context, cfg config.Config) (*ConfigManContext, error) {
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required for backend startup")
	}

	logger.DB.Info("DATABASE_URL detected; opening PostgreSQL connection")
	db, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}

	dataStore, err := store.NewStoreWithDB(ctx, db)
	if err != nil {
		_ = db.Close()
		return nil, err
	}

	return &ConfigManContext{
		Config: cfg,
		DB:     db,
		Store:  dataStore,
	}, nil
}

func (c *ConfigManContext) Close() error {
	if c == nil || c.DB == nil {
		return nil
	}
	return c.DB.Close()
}
