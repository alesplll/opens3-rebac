package repository

import (
	"context"
	"time"

	"github.com/alesplll/opens3-rebac/services/users/internal/model"
)

type UserRepository interface {
	Create(context.Context, *model.UserInfo, string, time.Time) (string, error)
	Get(context.Context, string) (*model.User, error)
	Update(context.Context, string, *string, *string) error
	UpdatePassword(context.Context, string, string) error
	LogPassword(context.Context, string, string) error
	Delete(context.Context, string) error
	GetUserCredentials(context.Context, string) (string, string, error)
}
