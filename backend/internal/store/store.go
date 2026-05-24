package store

import (
	"context"
	"database/sql"
	"strings"

	dbstore "config-man/backend/internal/db_store"
	"config-man/backend/model"
)

type Store struct {
	db           *dbstore.Store
	users        []model.User
	usersByID    map[string]model.User
	usersByEmail map[string]model.User
	groups       map[string]*model.Group
}

func NewStoreWithDB(ctx context.Context, db *sql.DB) (*Store, error) {
	client, err := dbstore.New(db)
	if err != nil {
		return nil, err
	}
	if err := client.InitSchema(ctx); err != nil {
		return nil, err
	}

	store := newStoreBase(client)
	hasProjects, err := client.HasProjects(ctx)
	if err != nil {
		return nil, err
	}
	if !hasProjects {
		seed := demoSeedData()
		if err := client.SaveSeedData(ctx, seed.groups, seed.projects, seed.configs, seed.reviews, seed.templates, seed.versions, seed.revisions, seed.audits); err != nil {
			return nil, err
		}
	}
	store.ensureDefaultSharedConfigs()
	return store, nil
}

func newStoreBase(db *dbstore.Store) *Store {
	store := &Store{
		db:           db,
		usersByID:    make(map[string]model.User),
		usersByEmail: make(map[string]model.User),
		groups:       make(map[string]*model.Group),
	}
	store.seedUsers()
	return store
}

func (s *Store) FindUserByEmail(email string) (model.User, bool) {
	user, ok := s.usersByEmail[strings.ToLower(strings.TrimSpace(email))]
	return user, ok
}

func (s *Store) FindUserByID(id string) (model.User, bool) {
	user, ok := s.usersByID[id]
	return user, ok
}
