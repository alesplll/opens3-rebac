package user

import (
	"context"
	"time"

	domainerrors "github.com/alesplll/opens3-rebac/services/users/internal/errors/domain_errors"
	"github.com/alesplll/opens3-rebac/services/users/internal/model"
	"github.com/alesplll/opens3-rebac/services/users/internal/validator"
	"github.com/alesplll/opens3-rebac/shared/pkg/kit/sys/validate"
)

func (s *userService) Create(ctx context.Context, userInfo model.UserInfo, password, passwordConfirm string) (string, error) {
	// UserInfo Validation
	if err := validate.Validate(
		ctx,
		validator.ValidateNotEmptyString(userInfo.Name, domainerrors.ErrNameRequired),
		validator.ValidateNotEmptyString(userInfo.Email, domainerrors.ErrEmailRequired),
		validator.ValidateEmailFromat(userInfo.Email),
	); err != nil {
		return "", err
	}

	// Password Validation + Hashing
	hashedPassword, err := s.validateAndHashPassword(ctx, password, passwordConfirm)
	if err != nil {
		return "", err
	}

	createdAt := time.Now()
	id, err := s.repo.Create(ctx, &userInfo, hashedPassword, createdAt)
	if err != nil {
		return "", err
	}

	return id, nil
}
