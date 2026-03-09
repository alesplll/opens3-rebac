package grpc

import (
	"context"

	"github.com/alesplll/opens3-rebac/services/auth/internal/model"
)

type UserClient interface {
	ValidateCredentials(context.Context, string, string) (model.ValidateCredentialsResult, error)
}
