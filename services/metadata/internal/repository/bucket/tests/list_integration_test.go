//go:build integration

package tests

import "time"

func (s *BucketRepositorySuite) TestList_ReturnsOnlyBucketsOfOwnerOrderedByCreatedAtDesc() {
	ownerID := s.ownerID("active")
	olderCreatedAt := time.Unix(1712345600, 0)
	newerCreatedAt := time.Unix(1712345700, 0)

	s.mustRecreateBucketWithCreatedAt(s.bucketID("older"), s.bucketName("older"), ownerID, olderCreatedAt)
	s.mustRecreateBucketWithCreatedAt(s.bucketID("newer"), s.bucketName("newer"), ownerID, newerCreatedAt)
	s.mustRecreateBucketWithCreatedAt(s.bucketID("foreign"), s.bucketName("foreign"), s.ownerID("foreign"), newerCreatedAt.Add(time.Minute))

	buckets, err := s.repo.List(s.Context(), ownerID)

	s.Require().NoError(err)
	s.Require().Len(buckets, 2)
	s.Require().Equal(s.bucketID("newer"), buckets[0].ID)
	s.Require().Equal(s.bucketID("older"), buckets[1].ID)
}

func (s *BucketRepositorySuite) TestList_ReturnsEmptySliceWhenOwnerHasNoBuckets() {
	buckets, err := s.repo.List(s.Context(), s.ownerID("missing"))

	s.Require().NoError(err)
	s.Require().Empty(buckets)
}
