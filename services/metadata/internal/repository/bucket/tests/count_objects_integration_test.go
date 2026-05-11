//go:build integration

package tests

func (s *BucketRepositorySuite) TestCountObjects_ReturnsOnlyObjectsOfRequestedBucket() {
	activeBucketID := s.bucketID("active")
	otherBucketID := s.bucketID("other")

	s.mustRecreateBucket(activeBucketID, s.bucketName("active"), s.ownerID("active"))
	s.mustRecreateBucket(otherBucketID, s.bucketName("other"), s.ownerID("other"))
	s.mustRecreateObject(s.objectID("first"), activeBucketID, s.objectKey("first"))
	s.mustRecreateObject(s.objectID("second"), activeBucketID, s.objectKey("second"))
	s.mustRecreateObject(s.objectID("foreign"), otherBucketID, s.objectKey("foreign"))

	count, err := s.repo.CountObjects(s.Context(), activeBucketID)

	s.Require().NoError(err)
	s.Require().Equal(int64(2), count)
}

func (s *BucketRepositorySuite) TestCountObjects_ReturnsZeroWhenBucketHasNoObjects() {
	s.mustRecreateBucket(s.bucketID("empty"), s.bucketName("empty"), s.ownerID("empty"))

	count, err := s.repo.CountObjects(s.Context(), s.bucketID("empty"))

	s.Require().NoError(err)
	s.Require().Zero(count)
}
