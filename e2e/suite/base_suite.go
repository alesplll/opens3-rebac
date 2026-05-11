package suite

import (
	"context"
	"strings"

	"github.com/alesplll/opens3-rebac/e2e/suite/config"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

type BaseSuite struct {
	suite.Suite

	ctx context.Context
	cfg *config.Config
}

func (s *BaseSuite) SetupSuite() {
	s.ctx = context.Background()
	s.cfg = config.MustLoad(s.T())
}

func (s *BaseSuite) Context() context.Context {
	return s.ctx
}

func (s *BaseSuite) Config() *config.Config {
	s.T().Helper()

	if s.cfg == nil {
		s.T().Fatal("e2e base suite config is not loaded")
	}

	return s.cfg
}

func (s *BaseSuite) MustRepoRoot() string {
	return config.MustRepoRoot(s.T())
}

func (s *BaseSuite) MustServiceMigrationDir(serviceName string) string {
	return config.MustServiceMigrationDir(s.T(), serviceName)
}

func (s *BaseSuite) FixtureUUID(parts ...string) string {
	s.T().Helper()

	return uuid.NewMD5(uuid.NameSpaceOID, []byte(strings.Join(parts, "/"))).String()
}
