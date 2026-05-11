package bucket

import (
	"github.com/alesplll/opens3-rebac/services/metadata/internal/repository"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/client/db"
)

const (
	bucketsTable = "buckets"
	objectsTable = "objects"

	idColumn        = "id"
	nameColumn      = "name"
	ownerIDColumn   = "owner_id"
	createdAtColumn = "created_at"
	bucketIDColumn  = "bucket_id"
)

type repo struct {
	db db.Client
}

func NewRepository(dbClient db.Client) repository.BucketRepository {
	return &repo{db: dbClient}
}
