package app

import (
	"context"
	"strings"

	"github.com/IBM/sarama"
	"github.com/alesplll/opens3-rebac/services/metadata/internal/config"
	metadataHandler "github.com/alesplll/opens3-rebac/services/metadata/internal/handler/metadata"
	bucketRepo "github.com/alesplll/opens3-rebac/services/metadata/internal/repository/bucket"
	objectRepo "github.com/alesplll/opens3-rebac/services/metadata/internal/repository/object"
	"github.com/alesplll/opens3-rebac/services/metadata/internal/repository"
	"github.com/alesplll/opens3-rebac/services/metadata/internal/service"
	bucketSvc "github.com/alesplll/opens3-rebac/services/metadata/internal/service/bucket"
	objectSvc "github.com/alesplll/opens3-rebac/services/metadata/internal/service/object"
	metadatav1 "github.com/alesplll/opens3-rebac/shared/pkg/go/metadata/v1"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/client/db"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/client/db/pg"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/client/db/transaction"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/closer"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/kafka"
	kafkaProducer "github.com/alesplll/opens3-rebac/shared/pkg/go-kit/kafka/producer"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/logger"
)

type serviceProvider struct {
	pgClient     db.Client
	txManager    db.TxManager
	saramaClient sarama.Client
	syncProducer sarama.SyncProducer

	bucketRepository repository.BucketRepository
	objectRepository repository.ObjectRepository

	objectDeletedProducer kafka.Producer
	bucketDeletedProducer kafka.Producer

	bucketService service.BucketService
	objectService service.ObjectService

	metadataHandler metadatav1.MetadataServiceServer
}

func newServiceProvider() *serviceProvider {
	return &serviceProvider{}
}

func (s *serviceProvider) PGClient(ctx context.Context) db.Client {
	if s.pgClient == nil {
		client, err := pg.NewPGClient(ctx, logger.Logger(), config.AppConfig().PG)
		if err != nil {
			panic(err)
		}

		if err := client.DB().Ping(ctx); err != nil {
			panic(err)
		}

		closer.AddNamed("PGClient", func(ctx context.Context) error {
			return client.Close()
		})

		s.pgClient = client
	}

	return s.pgClient
}

func (s *serviceProvider) TxManager(ctx context.Context) db.TxManager {
	if s.txManager == nil {
		s.txManager = transaction.NewTransactionManager(s.PGClient(ctx).DB())
	}

	return s.txManager
}

func (s *serviceProvider) SaramaClient(_ context.Context) sarama.Client {
	if s.saramaClient == nil {
		brokers := strings.Split(config.AppConfig().Kafka.BootstrapServers(), ",")

		cfg := sarama.NewConfig()
		cfg.Producer.Return.Successes = true
		cfg.Producer.Return.Errors = true

		client, err := sarama.NewClient(brokers, cfg)
		if err != nil {
			panic(err)
		}

		closer.AddNamed("SaramaClient", func(_ context.Context) error {
			return client.Close()
		})

		s.saramaClient = client
	}

	return s.saramaClient
}

func (s *serviceProvider) SyncProducer(ctx context.Context) sarama.SyncProducer {
	if s.syncProducer == nil {
		p, err := sarama.NewSyncProducerFromClient(s.SaramaClient(ctx))
		if err != nil {
			panic(err)
		}

		closer.AddNamed("SyncProducer", func(_ context.Context) error {
			return p.Close()
		})

		s.syncProducer = p
	}

	return s.syncProducer
}

func (s *serviceProvider) ObjectDeletedProducer(ctx context.Context) kafka.Producer {
	if s.objectDeletedProducer == nil {
		s.objectDeletedProducer = kafkaProducer.NewProducer(
			s.SyncProducer(ctx),
			config.AppConfig().Kafka.ObjectDeletedTopic(),
			logger.Logger(),
		)
	}

	return s.objectDeletedProducer
}

func (s *serviceProvider) BucketDeletedProducer(ctx context.Context) kafka.Producer {
	if s.bucketDeletedProducer == nil {
		s.bucketDeletedProducer = kafkaProducer.NewProducer(
			s.SyncProducer(ctx),
			config.AppConfig().Kafka.BucketDeletedTopic(),
			logger.Logger(),
		)
	}

	return s.bucketDeletedProducer
}

func (s *serviceProvider) BucketRepository(ctx context.Context) repository.BucketRepository {
	if s.bucketRepository == nil {
		s.bucketRepository = bucketRepo.NewRepository(s.PGClient(ctx))
	}

	return s.bucketRepository
}

func (s *serviceProvider) ObjectRepository(ctx context.Context) repository.ObjectRepository {
	if s.objectRepository == nil {
		s.objectRepository = objectRepo.NewRepository(s.PGClient(ctx))
	}

	return s.objectRepository
}

func (s *serviceProvider) BucketService(ctx context.Context) service.BucketService {
	if s.bucketService == nil {
		s.bucketService = bucketSvc.NewService(
			s.BucketRepository(ctx),
			s.TxManager(ctx),
			s.BucketDeletedProducer(ctx),
		)
	}

	return s.bucketService
}

func (s *serviceProvider) ObjectService(ctx context.Context) service.ObjectService {
	if s.objectService == nil {
		s.objectService = objectSvc.NewService(
			s.ObjectRepository(ctx),
			s.TxManager(ctx),
			s.ObjectDeletedProducer(ctx),
			s.PGClient(ctx),
			s.SaramaClient(ctx),
		)
	}

	return s.objectService
}

func (s *serviceProvider) MetadataHandler(ctx context.Context) metadatav1.MetadataServiceServer {
	if s.metadataHandler == nil {
		s.metadataHandler = metadataHandler.NewHandler(
			s.BucketService(ctx),
			s.ObjectService(ctx),
		)
	}

	return s.metadataHandler
}
