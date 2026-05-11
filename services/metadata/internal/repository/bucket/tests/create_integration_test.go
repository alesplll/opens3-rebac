//go:build integration

package tests

import (
	domainerrors "github.com/alesplll/opens3-rebac/services/metadata/internal/errors/domain_errors"
)

func (s *BucketRepositorySuite) TestCreate_CreatesBucket() {
	bucket, err := s.repo.Create(s.Context(), s.bucketName("active"), s.ownerID("active"))

	s.Require().NoError(err)
	s.Require().NotNil(bucket)
	s.Require().NotEmpty(bucket.ID)
	s.Require().Equal(s.bucketName("active"), bucket.Name)
	s.Require().Equal(s.ownerID("active"), bucket.OwnerID)
	s.Require().Equal(int64(1), s.MustCount("buckets"))
}

func (s *BucketRepositorySuite) TestCreate_ReturnsBucketAlreadyExistsWhenNameIsTaken() {
	s.mustRecreateBucket(s.bucketID("existing"), s.bucketName("existing"), s.ownerID("existing"))

	bucket, err := s.repo.Create(s.Context(), s.bucketName("existing"), s.ownerID("other"))

	s.Require().Nil(bucket)
	s.Require().ErrorIs(err, domainerrors.ErrBucketAlreadyExists)
	s.Require().Equal(int64(1), s.MustCount("buckets"))
}
