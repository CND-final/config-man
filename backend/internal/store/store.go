package store

import (
	"context"
	"database/sql"

	dbstore "config-man/backend/internal/db_store"
	"config-man/backend/model"
)

type Store struct {
	db     *dbstore.Store
	groups map[string]*model.Group
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

	hasUsers, err := client.HasUsers(ctx)
	if err != nil {
		return nil, err
	}
	if !hasUsers {
		if err := client.SaveSeedUsers(ctx, seedUserList()); err != nil {
			return nil, err
		}
	}

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
	return &Store{
		db:     db,
		groups: make(map[string]*model.Group),
	}
}

// FindUserByEmail looks up a user by email for Login. DB errors are treated
// as not-found; the caller (Login) will return Unauthorized in both cases.
func (s *Store) FindUserByEmail(email string) (model.User, bool) {
	user, ok, err := s.db.FindUserByEmail(context.Background(), email)
	if err != nil {
		return model.User{}, false
	}
	return user, ok
}

// FindUserByID looks up a user by ID for per-request auth token validation.
// DB errors are treated as not-found.
func (s *Store) FindUserByID(id string) (model.User, bool) {
	user, ok, err := s.db.FindUserByID(context.Background(), id)
	if err != nil {
		return model.User{}, false
	}
	return user, ok
}
