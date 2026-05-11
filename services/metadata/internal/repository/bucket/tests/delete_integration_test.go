//go:build integration

package tests

import (
	domainerrors "github.com/alesplll/opens3-rebac/services/metadata/internal/errors/domain_errors"
)

func (s *BucketRepositorySuite) TestDelete_DeletesBucketByID() {
	s.mustRecreateBucket(s.bucketID("active"), s.bucketName("active"), s.ownerID("active"))

	err := s.repo.Delete(s.Context(), s.bucketID("active"))

	s.Require().NoError(err)
	s.Require().Equal(int64(0), s.MustCount("buckets"))
}

func (s *BucketRepositorySuite) TestDelete_ReturnsBucketNotFoundWhenMissing() {
	err := s.repo.Delete(s.Context(), s.bucketID("missing"))

	s.Require().ErrorIs(err, domainerrors.ErrBucketNotFound)
}
