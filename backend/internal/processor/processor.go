package processor

import (
	"context"
	"log/slog"
)

type Processor struct {
	store *Store
	log   *slog.Logger
}

func New(log *slog.Logger) *Processor {
	return &Processor{
		store: NewStore(),
		log:   log,
	}
}

func NewWithDatabase(ctx context.Context, log *slog.Logger, databaseURL string) (*Processor, error) {
	store, err := NewStoreWithDatabase(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	return &Processor{store: store, log: log}, nil
}
