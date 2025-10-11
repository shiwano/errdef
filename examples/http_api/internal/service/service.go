package service

import (
	"context"
	"errors"
	"regexp"

	"github.com/shiwano/errdef"
	"github.com/shiwano/errdef/examples/http_api/internal/errdefs"
	"github.com/shiwano/errdef/examples/http_api/internal/repository"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

// Service defines the business logic interface for user operations.
type Service interface {
	GetUser(ctx context.Context, id string) (*repository.User, error)
	CreateUser(ctx context.Context, name, email string) (*repository.User, error)
	UpdateUser(ctx context.Context, id, name, email, currentUserID string) (*repository.User, error)
	DeleteUser(ctx context.Context, id, currentUserID string) error
}

// service implements the Service interface.
type service struct {
	repo repository.Repository
}

// New creates a new Service instance.
func New(repo repository.Repository) Service {
	return &service{
		repo: repo,
	}
}

func (s *service) GetUser(ctx context.Context, id string) (*repository.User, error) {
	user, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, errdefs.ErrNotFound) {
			return nil, err
		}
		return nil, errdefs.ErrDatabase.With(ctx, errdefs.UserID(id)).Wrap(err)
	}

	return user, nil
}

func (s *service) CreateUser(ctx context.Context, name, email string) (*repository.User, error) {
	if err := s.validateUserInput(ctx, name, email); err != nil {
		return nil, err
	}

	user := &repository.User{
		Name:  name,
		Email: email,
	}

	createdUser, err := s.repo.Create(ctx, user)
	if err != nil {
		if errors.Is(err, errdefs.ErrConflict) {
			return nil, err
		}
		return nil, errdefs.ErrDatabase.With(ctx, errdefs.Email(errdef.Redact(email))).Wrap(err)
	}

	return createdUser, nil
}

func (s *service) UpdateUser(ctx context.Context, id, name, email, currentUserID string) (*repository.User, error) {
	if err := s.validateUserInput(ctx, name, email); err != nil {
		return nil, err
	}

	if id != currentUserID {
		return nil, errdefs.ErrForbidden.With(ctx,
			errdefs.UserID(currentUserID),
			errdefs.ResourceType("user"),
			errdef.Details{"target_user_id": id},
		).New("cannot update another user's data")
	}

	user := &repository.User{
		ID:    id,
		Name:  name,
		Email: email,
	}

	if err := s.repo.Update(ctx, user); err != nil {
		if errors.Is(err, errdefs.ErrNotFound) {
			return nil, err
		}
		if errors.Is(err, errdefs.ErrConflict) {
			return nil, err
		}
		return nil, errdefs.ErrDatabase.With(ctx, errdefs.UserID(id)).Wrap(err)
	}

	return user, nil
}

func (s *service) DeleteUser(ctx context.Context, id, currentUserID string) error {
	if id != currentUserID {
		return errdefs.ErrForbidden.With(ctx,
			errdefs.UserID(currentUserID),
			errdefs.ResourceType("user"),
			errdef.Details{"target_user_id": id},
		).New("cannot delete another user's data")
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		if errors.Is(err, errdefs.ErrNotFound) {
			return err
		}
		return errdefs.ErrDatabase.With(ctx, errdefs.UserID(id)).Wrap(err)
	}

	return nil
}

func (s *service) validateUserInput(ctx context.Context, name, email string) error {
	validationErrs := make(map[string]string)

	if name == "" {
		validationErrs["name"] = "name is required"
	} else if len(name) > 100 {
		validationErrs["name"] = "name must be 100 characters or less"
	}

	if email == "" {
		validationErrs["email"] = "email is required"
	} else if !emailRegex.MatchString(email) {
		validationErrs["email"] = "email is invalid"
	}

	if len(validationErrs) > 0 {
		return errdefs.ErrValidation.With(ctx,
			errdefs.ValidationErrors(validationErrs),
		).New("validation failed")
	}

	return nil
}
