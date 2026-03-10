package user

import (
	"context"

	"github.com/alesplll/opens3-rebac/services/users/internal/validator"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/contextx/ipctx"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/logger"
	"golang.org/x/crypto/bcrypt"
)

func (s *userService) UpdatePassword(ctx context.Context, userID string, password, passwordConfirm string) error {
	// Input Validation + Hashing
	hashedPassword, err := s.validateAndHashPassword(ctx, password, passwordConfirm)
	if err != nil {
		return err
	}

	txErr := s.txManger.ReadCommitted(ctx, func(ctx context.Context) error {
		if err := s.repo.UpdatePassword(ctx, userID, hashedPassword); err != nil {
			return err
		}
		ip, ok := ipctx.ExtractIP(ctx)
		if !ok {
			logger.Error(ctx, "Failed to extract IP from context to user")
		}
		return s.repo.LogPassword(ctx, userID, ip)
	})

	return txErr
}

func (s *userService) validateAndHashPassword(ctx context.Context, password, passwordConfirm string) (string, error) {
	if err := validator.ValidatePassword(ctx, password, passwordConfirm); err != nil {
		return "", err
	}

	return s.hashPassword(password)
}

func (s *userService) hashPassword(password string) (string, error) {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	return string(hashedBytes), nil
}
