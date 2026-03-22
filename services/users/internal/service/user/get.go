package user

import (
	"context"

	"github.com/alesplll/opens3-rebac/services/users/internal/model"
)

func (s *userService) Get(ctx context.Context, id string) (*model.User, error) {
	user, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	return user, nil
}
