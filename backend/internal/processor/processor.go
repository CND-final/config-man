package processor

import (
	"fmt"

	"config-man/backend/internal/store"
)

type Processor struct {
	store *store.Store
}

func NewProcessor(dataStore *store.Store) (*Processor, error) {
	if dataStore == nil {
		return nil, fmt.Errorf("processor store is required")
	}
	return &Processor{store: dataStore}, nil
}

func New(dataStore *store.Store) *Processor {
	proc, err := NewProcessor(dataStore)
	if err != nil {
		return &Processor{store: store.NewStore()}
	}
	return proc
}

func NewInMemory() *Processor {
	return New(store.NewStore())
}
