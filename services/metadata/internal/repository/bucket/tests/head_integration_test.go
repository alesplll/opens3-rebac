//go:build integration

package tests

func (s *BucketRepositorySuite) TestHead_ReturnsBucketIdentityWhenExists() {
	s.mustRecreateBucket(s.bucketID("active"), s.bucketName("active"), s.ownerID("active"))

	exists, bucketID, ownerID, err := s.repo.Head(s.Context(), s.bucketName("active"))

	s.Require().NoError(err)
	s.Require().True(exists)
	s.Require().Equal(s.bucketID("active"), bucketID)
	s.Require().Equal(s.ownerID("active"), ownerID)
}

func (s *BucketRepositorySuite) TestHead_ReturnsFalseWhenMissing() {
	exists, bucketID, ownerID, err := s.repo.Head(s.Context(), s.bucketName("missing"))

	s.Require().NoError(err)
	s.Require().False(exists)
	s.Require().Empty(bucketID)
	s.Require().Empty(ownerID)
}
