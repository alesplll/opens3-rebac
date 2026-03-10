package user

import (
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/client/db"
	"github.com/alesplll/opens3-rebac/services/users/internal/repository"
)

const (
	usersTableName = "users"

	idColumn        = "id"
	nameColumn      = "name"
	emailColumn     = "email"
	passwordColumn  = "password"
	createdAtColumn = "created_at"
	updatedAtColumn = "updated_at"

	passwordLogsTableName = "password_change_logs"

	passwordLogsIdColumn        = "id"
	passwordLogsUserIdColumn    = "user_id"
	passwordLogsChangedAtColumn = "changed_at"
	passwordLogsIpAddressColumn = "ip_address"
)

type repo struct {
	db db.Client
}

func NewRepository(db db.Client) repository.UserRepository {
	return &repo{db: db}
}
