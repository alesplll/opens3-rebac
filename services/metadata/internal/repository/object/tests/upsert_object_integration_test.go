//go:build integration

package tests

import (
	domainerrors "github.com/alesplll/opens3-rebac/services/metadata/internal/errors/domain_errors"
)

func (s *ObjectRepositorySuite) TestUpsertObject_CreatesObjectForExistingBucket() {
	s.mustRecreateBucket(s.bucketID(), s.bucketName(), s.ownerID())

	objectKey := s.objectKey(objectRoleActive)
	objectID, err := s.repo.UpsertObject(s.Context(), s.bucketName(), objectKey)

	s.Require().NoError(err)
	s.Require().NotEmpty(objectID)
	s.Require().Equal(int64(1), s.MustCount("objects"))

	var gotBucketID string
	var gotKey string
	err = s.MustQueryRow(`
		SELECT bucket_id, key
		FROM objects
		WHERE id = $1
	`, objectID).Scan(&gotBucketID, &gotKey)
	s.Require().NoError(err)
	s.Require().Equal(s.bucketID(), gotBucketID)
	s.Require().Equal(objectKey, gotKey)
}

func (s *ObjectRepositorySuite) TestUpsertObject_ReturnsBucketNotFoundWhenBucketDoesNotExist() {
	objectID, err := s.repo.UpsertObject(s.Context(), s.bucketName(), s.objectKey(objectRoleActive))

	s.Require().Empty(objectID)
	s.Require().ErrorIs(err, domainerrors.ErrBucketNotFound)
	s.Require().Equal(int64(0), s.MustCount("objects"))
}
