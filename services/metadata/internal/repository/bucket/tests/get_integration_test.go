//go:build integration

package tests

import (
	domainerrors "github.com/alesplll/opens3-rebac/services/metadata/internal/errors/domain_errors"
)

func (s *BucketRepositorySuite) TestGet_ReturnsBucketByName() {
	s.mustRecreateBucket(s.bucketID("active"), s.bucketName("active"), s.ownerID("active"))

	bucket, err := s.repo.Get(s.Context(), s.bucketName("active"))

	s.Require().NoError(err)
	s.Require().NotNil(bucket)
	s.Require().Equal(s.bucketID("active"), bucket.ID)
	s.Require().Equal(s.bucketName("active"), bucket.Name)
	s.Require().Equal(s.ownerID("active"), bucket.OwnerID)
}

func (s *BucketRepositorySuite) TestGet_ReturnsBucketNotFoundWhenMissing() {
	bucket, err := s.repo.Get(s.Context(), s.bucketName("missing"))

	s.Require().Nil(bucket)
	s.Require().ErrorIs(err, domainerrors.ErrBucketNotFound)
}
