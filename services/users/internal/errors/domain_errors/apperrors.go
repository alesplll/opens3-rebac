package domainerrors

import (
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/sys"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/sys/codes"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/sys/validate"
)

var (
	// Resource errors (NotFound)
	ErrUserNotFound = sys.NewCommonError("user not found", codes.NotFound)

	// Verify errors
	ErrFailedToVerify = sys.NewCommonError("failed to verify user", codes.Unauthenticated)

	// Permission errors
	ErrNoPermission = sys.NewCommonError("user have no permission to to this action", codes.PermissionDenied)

	// Conflict errors (AlreadyExists)
	ErrEmailAlreadyExists = sys.NewCommonError("email already exists", codes.AlreadyExists)

	// Validation errors - General (InvalidArgument)
	ErrInvalidInput = validate.NewValidationErrors("invalid input")

	// Validation errors - Required fields (InvalidArgument)
	ErrNameRequired      = validate.NewValidationErrors("name is required")
	ErrEmailRequired     = validate.NewValidationErrors("email is required")
	ErrPasswordRequired  = validate.NewValidationErrors("password is required")
	ErrNoChangesProvided = validate.NewValidationErrors("no changes provided")

	// Validation errors - Format (InvalidArgument)
	ErrInvalidEmailFormat = validate.NewValidationErrors("invalid email format")

	// Validation errors - Password (InvalidArgument)
	ErrPasswordMismatch = validate.NewValidationErrors("passwords do not match")
	ErrPasswordTooShort = validate.NewValidationErrors("password must be at least 5 characters long")

	// Internal errors
	ErrInternal = sys.NewCommonError("internal error", codes.Internal)
)
