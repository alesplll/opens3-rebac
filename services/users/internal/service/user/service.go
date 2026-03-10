package user

import (
	"github.com/alesplll/opens3-rebac/services/users/internal/repository"
	"github.com/alesplll/opens3-rebac/services/users/internal/service"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/client/db"
)

type userService struct {
	repo     repository.UserRepository
	txManger db.TxManager
}

func NewService(
	repo repository.UserRepository,
	txManger db.TxManager,
) service.UserService {
	return &userService{
		repo:     repo,
		txManger: txManger,
	}
}
