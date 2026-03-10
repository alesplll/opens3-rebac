package validator

import (
	"context"

	domainerrors "github.com/alesplll/opens3-rebac/services/users/internal/errors/domain_errors"
	"github.com/alesplll/opens3-rebac/services/users/internal/utils"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/sys/validate"
)

func ValidateNotEmptyString(text string, err error) validate.Condition {
	return func(ctx context.Context) error {
		if text == "" {
			return err
		}

		return nil
	}
}

func ValidateNotEmptyPointerToString(text *string, err error) validate.Condition {
	return func(ctx context.Context) error {
		if text != nil && *text == "" {
			return err
		}

		return nil
	}
}

func ValidateEmailFromat(email string) validate.Condition {
	return func(ctx context.Context) error {
		if !utils.IsValidEmail(email) {
			return domainerrors.ErrInvalidEmailFormat
		}

		return nil
	}
}

func ValidateEmailFromatPointer(email *string) validate.Condition {
	return func(ctx context.Context) error {
		if email != nil && !utils.IsValidEmail(*email) {
			return domainerrors.ErrInvalidEmailFormat
		}

		return nil
	}
}

func ValidatePasswordMismatch(password, passwordConfirm string) validate.Condition {
	return func(ctx context.Context) error {
		if password != passwordConfirm {
			return domainerrors.ErrPasswordMismatch
		}

		return nil
	}
}

func ValidatePasswordTooShort(password string) validate.Condition {
	return func(ctx context.Context) error {
		if len(password) < 5 {
			return domainerrors.ErrPasswordTooShort
		}

		return nil
	}
}
