package user

import (
	"github.com/alesplll/opens3-rebac/services/users/internal/service"
	desc "github.com/alesplll/opens3-rebac/shared/pkg/go/user/v1"
)

type handler struct {
	desc.UnimplementedUserV1Server
	service service.UserService
}

func NewHandler(service service.UserService) desc.UserV1Server {
	return &handler{
		service: service,
	}
}
