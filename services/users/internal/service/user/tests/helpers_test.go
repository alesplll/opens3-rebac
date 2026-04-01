package tests

import (
	"testing"

	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/sys/validate"
	"github.com/stretchr/testify/require"
)

func strPtr(value string) *string {
	return &value
}

func requireValidationError(t *testing.T, err error) {
	t.Helper()

	require.Error(t, err)
	require.True(t, validate.IsValidationError(err))
}
