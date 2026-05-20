package processor

import "config-man/backend/internal/store"

type Processor struct {
	store *store.Store
}

func New(dataStore *store.Store) *Processor {
	return &Processor{store: dataStore}
}

func NewInMemory() *Processor {
	return New(store.NewStore())
}
