package repository

import (
	"context"
	"strconv"
	"sync"

	"github.com/shiwano/errdef"
	"github.com/shiwano/errdef/examples/http_api/internal/errdefs"
)

// User represents a user entity.
type User struct {
	ID    string
	Name  string
	Email string
}

// Repository defines the data access interface for users.
type Repository interface {
	FindByID(ctx context.Context, id string) (*User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)
	Create(ctx context.Context, user *User) (*User, error)
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id string) error
}

// inMemoryRepository is an in-memory implementation of Repository for demonstration.
type inMemoryRepository struct {
	mu      sync.RWMutex
	users   map[string]*User
	nextID  int
	failing bool // Simulates database failures when true
}

// NewInMemory creates a new in-memory repository with some seed data.
func NewInMemory() Repository {
	return &inMemoryRepository{
		users: map[string]*User{
			"1": {ID: "1", Name: "Alice", Email: "alice@example.com"},
			"2": {ID: "2", Name: "Bob", Email: "bob@example.com"},
		},
		nextID:  3,
		failing: false,
	}
}

func (r *inMemoryRepository) FindByID(ctx context.Context, id string) (*User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.failing {
		return nil, errdefs.ErrDatabase.With(ctx, errdefs.UserID(id)).New("database connection failed")
	}

	user, ok := r.users[id]
	if !ok {
		return nil, errdefs.ErrNotFound.With(ctx, errdefs.UserID(id)).New("user not found")
	}

	return user, nil
}

func (r *inMemoryRepository) FindByEmail(ctx context.Context, email string) (*User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.failing {
		return nil, errdefs.ErrDatabase.With(ctx).New("database connection failed")
	}

	for _, user := range r.users {
		if user.Email == email {
			return user, nil
		}
	}

	return nil, errdefs.ErrNotFound.With(ctx).New("user not found")
}

func (r *inMemoryRepository) Create(ctx context.Context, user *User) (*User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.failing {
		return nil, errdefs.ErrDatabase.With(ctx).New("database connection failed")
	}

	for _, existing := range r.users {
		if existing.Email == user.Email {
			return nil, errdefs.ErrConflict.With(ctx,
				errdefs.Email(errdef.Redact(user.Email)),
			).New("email already exists")
		}
	}

	user.ID = r.generateID()
	r.users[user.ID] = user
	return user, nil
}

func (r *inMemoryRepository) Update(ctx context.Context, user *User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.failing {
		return errdefs.ErrDatabase.With(ctx, errdefs.UserID(user.ID)).New("database connection failed")
	}

	if _, ok := r.users[user.ID]; !ok {
		return errdefs.ErrNotFound.With(ctx, errdefs.UserID(user.ID)).New("user not found")
	}

	for id, existing := range r.users {
		if id != user.ID && existing.Email == user.Email {
			return errdefs.ErrConflict.With(ctx,
				errdefs.UserID(user.ID),
				errdefs.Email(errdef.Redact(user.Email)),
			).New("email already exists")
		}
	}

	r.users[user.ID] = user
	return nil
}

func (r *inMemoryRepository) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.failing {
		return errdefs.ErrDatabase.With(ctx, errdefs.UserID(id)).New("database connection failed")
	}

	if _, ok := r.users[id]; !ok {
		return errdefs.ErrNotFound.With(ctx, errdefs.UserID(id)).New("user not found")
	}

	delete(r.users, id)
	return nil
}

func (r *inMemoryRepository) generateID() string {
	id := strconv.Itoa(r.nextID)
	r.nextID++
	return id
}
