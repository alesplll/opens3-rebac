package domainerrors

import (
	"github.com/alesplll/opens3-rebac/shared/pkg/kit/sys"
	"github.com/alesplll/opens3-rebac/shared/pkg/kit/sys/codes"
)

var (
	// Authentication errors (Unauthenticated)
	ErrInvalidRefreshToken    = sys.NewCommonError("invalid refresh token", codes.Unauthenticated)
	ErrInvalidEmailOrPassword = sys.NewCommonError("invalid email or password", codes.Unauthenticated)

	// Rate limiting errors (ResourceExhausted)
	ErrTooManyAttempts = sys.NewCommonError("too many attempts", codes.ResourceExhausted)
)
