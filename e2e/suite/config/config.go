package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/alesplll/opens3-rebac/e2e/suite/config/env"
	"github.com/joho/godotenv"
)

var appConfig *Config

type Config struct {
	MetadataPG PGConfig
}

func Load(path ...string) error {
	envPaths := path
	if len(envPaths) == 0 {
		envPaths = []string{defaultEnvFilePath()}
	}

	err := godotenv.Overload(envPaths...)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	metadataPGCfg, err := env.NewMetadataPGConfig()
	if err != nil {
		return err
	}

	appConfig = &Config{
		MetadataPG: metadataPGCfg,
	}

	return nil
}

func MustLoad(tb testing.TB, path ...string) *Config {
	tb.Helper()

	if err := Load(path...); err != nil {
		tb.Fatalf("load e2e config: %v", err)
	}

	return AppConfig()
}

func AppConfig() *Config {
	if appConfig == nil {
		return nil
	}

	return appConfig
}

func MustRepoRoot(tb testing.TB) string {
	tb.Helper()

	root, err := repoRoot()
	if err != nil {
		tb.Fatalf("resolve repository root: %v", err)
	}

	return root
}

func MustServiceMigrationDir(tb testing.TB, serviceName string) string {
	tb.Helper()

	root := MustRepoRoot(tb)
	return filepath.Join(root, "services", serviceName, "migrations")
}

func defaultEnvFilePath() string {
	root, err := repoRoot()
	if err != nil {
		return filepath.Join("e2e", ".env")
	}

	return filepath.Join(root, "e2e", ".env")
}

func repoRoot() (string, error) {
	_, fileName, _, ok := runtime.Caller(0)
	if !ok {
		return "", os.ErrNotExist
	}

	return filepath.Clean(filepath.Join(filepath.Dir(fileName), "..", "..", "..")), nil
}
