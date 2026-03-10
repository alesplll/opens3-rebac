package user

import (
	"context"

	domainerrors "github.com/alesplll/opens3-rebac/services/users/internal/errors/domain_errors"
	"github.com/alesplll/opens3-rebac/services/users/internal/validator"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/sys/validate"
)

func (s *userService) Update(ctx context.Context, userID string, name, email *string) error {
	// Input validation
	if err := validate.Validate(
		ctx,
		validator.ValidateNotEmptyPointerToString(name, domainerrors.ErrNameRequired),
		validator.ValidateNotEmptyPointerToString(email, domainerrors.ErrEmailRequired),
		validator.ValidateEmailFromatPointer(email),
	); err != nil {
		return err
	}

	return s.repo.Update(ctx, userID, name, email)
}
