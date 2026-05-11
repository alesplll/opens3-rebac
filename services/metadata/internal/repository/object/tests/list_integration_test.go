//go:build integration

package tests

func (s *ObjectRepositorySuite) TestList_ReturnsOnlyActiveCommittedBlobVersions() {
	s.mustRecreateListFixture()

	items, nextToken, isTruncated, err := s.repo.List(s.Context(), s.bucketName(), s.objectPrefix(), "", 100)

	s.Require().NoError(err)
	s.Require().False(isTruncated)
	s.Require().Empty(nextToken)
	s.Require().Len(items, 1)
	s.Require().Equal(s.objectID(objectRoleActive), items[0].ObjectID)
	s.Require().Equal(s.versionID(objectRoleActive), items[0].VersionID)
	s.Require().Equal(s.objectKey(objectRoleActive), items[0].Key)
	s.Require().Equal(int64(3), s.MustCount("objects"))
	s.Require().Equal(int64(3), s.MustCount("versions"))
}
