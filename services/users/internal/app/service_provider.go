package app

import (
	"context"

	"github.com/alesplll/opens3-rebac/services/users/internal/config"
	userHandler "github.com/alesplll/opens3-rebac/services/users/internal/handler/user"
	"github.com/alesplll/opens3-rebac/services/users/internal/repository"
	userRepository "github.com/alesplll/opens3-rebac/services/users/internal/repository/user"
	"github.com/alesplll/opens3-rebac/services/users/internal/service"
	userService "github.com/alesplll/opens3-rebac/services/users/internal/service/user"
	desc "github.com/alesplll/opens3-rebac/shared/pkg/user/v1"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/client/db"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/client/db/pg"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/client/db/transaction"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/closer"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/logger"
)

type serviceProvider struct {
	pgClient  db.Client
	txManager db.TxManager

	userRepository repository.UserRepository

	userService service.UserService

	userHandler desc.UserV1Server
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

func (s *serviceProvider) UserRepository(ctx context.Context) repository.UserRepository {
	if s.userRepository == nil {
		s.userRepository = userRepository.NewRepository(s.PGClient(ctx))
	}

	return s.userRepository
}

func (s *serviceProvider) TxManager(ctx context.Context) db.TxManager {
	if s.txManager == nil {
		s.txManager = transaction.NewTransactionManager(s.PGClient(ctx).DB())
	}

	return s.txManager
}

func (s *serviceProvider) UserService(ctx context.Context) service.UserService {
	if s.userService == nil {
		s.userService = userService.NewService(s.UserRepository(ctx), s.TxManager(ctx))
	}

	return s.userService
}

func (s *serviceProvider) UserHandler(ctx context.Context) desc.UserV1Server {
	if s.userHandler == nil {
		s.userHandler = userHandler.NewHandler(s.UserService(ctx))
	}

	return s.userHandler
}
