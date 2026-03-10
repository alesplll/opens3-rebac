package service

import (
	"context"

	"github.com/alesplll/opens3-rebac/services/users/internal/model"
)

type UserService interface {
	Create(context.Context, model.UserInfo, string, string) (string, error)
	Get(context.Context, string) (*model.User, error)
	Update(context.Context, string, *string, *string) error
	UpdatePassword(context.Context, string, string, string) error
	Delete(context.Context, string) error
	ValidateCredentials(context.Context, string, string) (bool, string)
}
