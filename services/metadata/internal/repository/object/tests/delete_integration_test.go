//go:build integration

package tests

func (s *ObjectRepositorySuite) TestDelete_AllowsNullCurrentVersionBlob() {
	s.mustRecreateObjectWithoutCurrentVersion(s.objectID(objectRoleActive), s.objectKey(objectRoleActive))
	s.Require().Equal(int64(1), s.MustCount("objects"))

	objectID, blobID, err := s.repo.Delete(s.Context(), s.bucketName(), s.objectKey(objectRoleActive))

	s.Require().NoError(err)
	s.Require().Equal(s.objectID(objectRoleActive), objectID)
	s.Require().Empty(blobID)
	s.Require().Equal(int64(0), s.MustCount("objects"))
}
