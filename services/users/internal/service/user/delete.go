package user

import (
	"context"
)

func (s *userService) Delete(ctx context.Context, userID string) error {
	return s.repo.Delete(ctx, userID)
}
