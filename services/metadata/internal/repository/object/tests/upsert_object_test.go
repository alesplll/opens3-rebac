package tests

import (
	"testing"

	domainerrors "github.com/alesplll/opens3-rebac/services/metadata/internal/errors/domain_errors"
	"github.com/stretchr/testify/require"
)

func TestUpsertObject_ReturnsBucketNotFoundWhenBucketDoesNotExist(t *testing.T) {
	ctx, _, cleanup, repo := newRepository(t)
	defer cleanup()

	objectID, err := repo.UpsertObject(ctx, "missing-bucket", "photos/cat.jpg")

	require.Empty(t, objectID)
	require.ErrorIs(t, err, domainerrors.ErrBucketNotFound)
}
