package repository

import (
	"context"
)

type AuthRepository interface {
	IncrementLoginAttempts(ctx context.Context, email string) (int8, error)
	ResetLoginAttempts(ctx context.Context, email string) error
	GetLoginAttempts(ctx context.Context, email string) (int8, error)
}
