package object

import (
	"github.com/IBM/sarama"
	"github.com/alesplll/opens3-rebac/services/metadata/internal/repository"
	"github.com/alesplll/opens3-rebac/services/metadata/internal/service"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/client/db"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/kafka"
)

type objectService struct {
	repo          repository.ObjectRepository
	txManager     db.TxManager
	objectDeleted kafka.Producer
	pgClient      db.Client
	saramaClient  sarama.Client
}

func NewService(
	repo repository.ObjectRepository,
	txManager db.TxManager,
	objectDeleted kafka.Producer,
	pgClient db.Client,
	saramaClient sarama.Client,
) service.ObjectService {
	return &objectService{
		repo:          repo,
		txManager:     txManager,
		objectDeleted: objectDeleted,
		pgClient:      pgClient,
		saramaClient:  saramaClient,
	}
}
