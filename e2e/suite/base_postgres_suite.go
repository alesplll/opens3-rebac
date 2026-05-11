package suite

import (
	"fmt"

	"github.com/alesplll/opens3-rebac/e2e/suite/config"
	e2epostgres "github.com/alesplll/opens3-rebac/e2e/suite/postgres"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type BasePostgresSuite struct {
	BaseSuite

	pgConfig      config.PGConfig
	pool          *pgxpool.Pool
	migrationDir  string
	cleanupTables []string
}

type PostgresSuiteOptions struct {
	MigrationDir    string
	CleanupTables   []string
	ResolvePGConfig func(cfg *config.Config) config.PGConfig
}

func (s *BasePostgresSuite) SetupPostgresSuite(options PostgresSuiteOptions) {
	s.BaseSuite.SetupSuite()
	if options.ResolvePGConfig == nil {
		s.T().Fatal("postgres suite requires ResolvePGConfig")
	}
	if options.MigrationDir == "" {
		s.T().Fatal("postgres suite requires MigrationDir")
	}
	if len(options.CleanupTables) == 0 {
		s.T().Fatal("postgres suite requires CleanupTables")
	}

	s.pgConfig = options.ResolvePGConfig(s.Config())
	if s.pgConfig == nil {
		s.T().Fatal("postgres suite resolved nil PG config")
	}

	s.migrationDir = options.MigrationDir
	s.cleanupTables = options.CleanupTables

	s.pool = e2epostgres.MustConnect(s.T(), s.Context(), s.pgConfig)
	e2epostgres.MustResetPublicSchema(s.T(), s.Context(), s.pool)
	e2epostgres.MustApplyMigrations(s.T(), s.Context(), s.pool, s.migrationDir)
}

func (s *BasePostgresSuite) TearDownSuite() {
	if s.pool != nil {
		s.pool.Close()
	}
}

func (s *BasePostgresSuite) SetupTest() {
	e2epostgres.MustFullCleanup(s.T(), s.Context(), s.pool, s.cleanupTables...)
}

func (s *BasePostgresSuite) PGConfig() config.PGConfig {
	s.T().Helper()

	if s.pgConfig == nil {
		s.T().Fatal("postgres suite PG config is not initialized")
	}

	return s.pgConfig
}

func (s *BasePostgresSuite) Pool() *pgxpool.Pool {
	s.T().Helper()

	if s.pool == nil {
		s.T().Fatal("postgres suite pool is not initialized")
	}

	return s.pool
}

func (s *BasePostgresSuite) MigrationDir() string {
	s.T().Helper()

	if s.migrationDir == "" {
		s.T().Fatal("postgres suite migration dir is not initialized")
	}

	return s.migrationDir
}

func (s *BasePostgresSuite) MustExec(query string, args ...any) {
	s.T().Helper()

	e2epostgres.MustExec(s.T(), s.Context(), s.Pool(), query, args...)
}

func (s *BasePostgresSuite) MustQueryRow(query string, args ...any) pgx.Row {
	s.T().Helper()

	return e2epostgres.MustQueryRow(s.T(), s.Context(), s.Pool(), query, args...)
}

func (s *BasePostgresSuite) MustCount(tableName string) int64 {
	s.T().Helper()

	return s.MustCountWhere(tableName, "")
}

func (s *BasePostgresSuite) MustCountWhere(tableName, whereClause string, args ...any) int64 {
	s.T().Helper()

	query := fmt.Sprintf("SELECT count(*) FROM %s", tableName)
	if whereClause != "" {
		query = fmt.Sprintf("%s WHERE %s", query, whereClause)
	}

	var count int64
	err := s.MustQueryRow(query, args...).Scan(&count)
	s.Require().NoError(err)

	return count
}
