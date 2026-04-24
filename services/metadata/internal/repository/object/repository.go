package object

import (
	"github.com/alesplll/opens3-rebac/services/metadata/internal/repository"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/client/db"
)

const (
	objectsTable  = "objects"
	versionsTable = "versions"
)

type repo struct {
	db db.Client
}

func NewRepository(dbClient db.Client) repository.ObjectRepository {
	return &repo{db: dbClient}
}
