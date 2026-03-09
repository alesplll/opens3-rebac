package validator

import (
	domainerrors "github.com/alesplll/opens3-rebac/services/users/internal/errors/domain_errors"
	"github.com/alesplll/opens3-rebac/shared/pkg/kit/sys/validate"
	"golang.org/x/net/context"
)

func ValidatePassword(ctx context.Context, password, passwordConfirm string) error {
	return validate.Validate(
		ctx,
		ValidateNotEmptyString(password, domainerrors.ErrPasswordRequired),
		ValidatePasswordTooShort(password),
		ValidatePasswordMismatch(password, passwordConfirm),
	)
}
