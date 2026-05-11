//go:build integration

package tests

import (
	"context"
	"testing"

	e2esuite "github.com/alesplll/opens3-rebac/e2e/suite"
	e2econfig "github.com/alesplll/opens3-rebac/e2e/suite/config"
	"github.com/alesplll/opens3-rebac/services/metadata/internal/repository"
	bucketRepo "github.com/alesplll/opens3-rebac/services/metadata/internal/repository/bucket"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/client/db"
	pgclient "github.com/alesplll/opens3-rebac/shared/pkg/go-kit/client/db/pg"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

type testLogger struct{}

func (testLogger) Debug(context.Context, string, ...zap.Field) {}

type BucketRepositorySuite struct {
	e2esuite.BasePostgresSuite

	client db.Client
	repo   repository.BucketRepository
}

func TestBucketRepositorySuite(t *testing.T) {
	suite.Run(t, new(BucketRepositorySuite))
}

func (s *BucketRepositorySuite) SetupSuite() {
	s.SetupPostgresSuite(e2esuite.PostgresSuiteOptions{
		MigrationDir:  s.MustServiceMigrationDir("metadata"),
		CleanupTables: []string{"versions", "objects", "buckets"},
		ResolvePGConfig: func(cfg *e2econfig.Config) e2econfig.PGConfig {
			return cfg.MetadataPG
		},
	})

	client, err := pgclient.NewPGClient(s.Context(), testLogger{}, s.PGConfig())
	s.Require().NoError(err)

	s.client = client
	s.repo = bucketRepo.NewRepository(client)
}

func (s *BucketRepositorySuite) TearDownSuite() {
	if s.client != nil {
		_ = s.client.Close()
	}

	s.BasePostgresSuite.TearDownSuite()
}
