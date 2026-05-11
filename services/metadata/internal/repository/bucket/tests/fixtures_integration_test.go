//go:build integration

package tests

import "time"

func (s *BucketRepositorySuite) bucketID(role string) string {
	s.T().Helper()

	return s.FixtureUUID(s.T().Name(), "bucket", role)
}

func (s *BucketRepositorySuite) ownerID(role string) string {
	s.T().Helper()

	return s.FixtureUUID(s.T().Name(), "owner", role)
}

func (s *BucketRepositorySuite) objectID(role string) string {
	s.T().Helper()

	return s.FixtureUUID(s.T().Name(), "object", role)
}

func (s *BucketRepositorySuite) bucketName(role string) string {
	s.T().Helper()

	return s.T().Name() + "-" + role + "-bucket"
}

func (s *BucketRepositorySuite) objectKey(role string) string {
	s.T().Helper()

	return s.T().Name() + "/" + role + ".txt"
}

func (s *BucketRepositorySuite) mustRecreateBucket(id, name, ownerID string) {
	s.T().Helper()

	s.MustExec(`
		DELETE FROM buckets
		WHERE id = $1 OR name = $2
	`, id, name)

	s.MustExec(`
		INSERT INTO buckets (id, name, owner_id)
		VALUES ($1, $2, $3)
	`, id, name, ownerID)
}

func (s *BucketRepositorySuite) mustRecreateBucketWithCreatedAt(id, name, ownerID string, createdAt time.Time) {
	s.T().Helper()

	s.mustRecreateBucket(id, name, ownerID)
	s.MustExec(`
		UPDATE buckets
		SET created_at = $2
		WHERE id = $1
	`, id, createdAt)
}

func (s *BucketRepositorySuite) mustRecreateObject(id, bucketID, key string) {
	s.T().Helper()

	s.MustExec(`
		DELETE FROM objects
		WHERE id = $1 OR (bucket_id = $2 AND key = $3)
	`, id, bucketID, key)

	s.MustExec(`
		INSERT INTO objects (id, bucket_id, key, status)
		VALUES ($1, $2, $3, 'active')
	`, id, bucketID, key)
}
