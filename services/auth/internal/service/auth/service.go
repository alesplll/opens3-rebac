package auth

import (
	"github.com/alesplll/opens3-rebac/services/auth/internal/client/grpc"
	"github.com/alesplll/opens3-rebac/services/auth/internal/config"
	"github.com/alesplll/opens3-rebac/services/auth/internal/repository"
	"github.com/alesplll/opens3-rebac/services/auth/internal/service"
	"github.com/alesplll/opens3-rebac/shared/pkg/kit/tokens"
)

type authService struct {
	userClient     grpc.UserClient
	tokenService   tokens.TokenService
	repository     repository.AuthRepository
	securityConfig config.SecurityConfig
}

func NewService(userClient grpc.UserClient, tokenService tokens.TokenService, repository repository.AuthRepository) service.AuthService {
	return &authService{
		userClient:   userClient,
		tokenService: tokenService,
		repository:   repository,
	}
}
