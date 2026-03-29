package auth

import (
	"github.com/alesplll/opens3-rebac/services/auth/internal/service"
	desc "github.com/alesplll/opens3-rebac/shared/pkg/go/auth/v1"
)

type authHandler struct {
	desc.UnimplementedAuthV1Server
	service service.AuthService
}

func NewHandler(service service.AuthService) desc.AuthV1Server {
	return &authHandler{
		service: service,
	}
}
