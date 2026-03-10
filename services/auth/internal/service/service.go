package service

import "context"

type AuthService interface {
	Login(context.Context, string, string) (string, error)
	GetRefreshToken(context.Context, string) (string, error)
	GetAccessToken(context.Context, string) (string, error)
	ValidateToken(context.Context, string) error
}
