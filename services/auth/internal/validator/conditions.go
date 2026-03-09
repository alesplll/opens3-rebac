package conditions

import (
	"context"

	"github.com/alesplll/opens3-rebac/shared/pkg/kit/sys/validate"
)

func ValidateNotEmptyEmailAndPassword(email, password string) validate.Condition {
	return func(ctx context.Context) error {
		if email == "" || password == "" {
			return validate.NewValidationErrors("empty credentials")
		}

		return nil
	}
}
