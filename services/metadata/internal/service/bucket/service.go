package bucket

import (
	"github.com/alesplll/opens3-rebac/services/metadata/internal/repository"
	"github.com/alesplll/opens3-rebac/services/metadata/internal/service"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/client/db"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/kafka"
)

type bucketService struct {
	repo          repository.BucketRepository
	txManager     db.TxManager
	bucketDeleted kafka.Producer
}

func NewService(repo repository.BucketRepository, txManager db.TxManager, bucketDeleted kafka.Producer) service.BucketService {
	return &bucketService{
		repo:          repo,
		txManager:     txManager,
		bucketDeleted: bucketDeleted,
	}
}
