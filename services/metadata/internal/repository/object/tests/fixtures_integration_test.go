//go:build integration

package tests

import (
	"fmt"
)

const (
	objectRoleActive  = "active"
	objectRoleOther   = "other"
	objectRoleDeleted = "deleted"
)

func (s *ObjectRepositorySuite) bucketID() string {
	s.T().Helper()

	return s.FixtureUUID(s.T().Name(), "bucket")
}

func (s *ObjectRepositorySuite) ownerID() string {
	s.T().Helper()

	return s.FixtureUUID(s.T().Name(), "owner")
}

func (s *ObjectRepositorySuite) bucketName() string {
	s.T().Helper()

	return s.T().Name() + "-bucket"
}

func (s *ObjectRepositorySuite) objectID(role string) string {
	s.T().Helper()

	return s.FixtureUUID(s.T().Name(), "object", role)
}

func (s *ObjectRepositorySuite) versionID(role string) string {
	s.T().Helper()

	return s.FixtureUUID(s.T().Name(), "version", role)
}

func (s *ObjectRepositorySuite) blobID(role string) string {
	s.T().Helper()

	return s.FixtureUUID(s.T().Name(), "blob", role)
}

func (s *ObjectRepositorySuite) objectKey(role string) string {
	s.T().Helper()

	return fmt.Sprintf("%s/%s.txt", s.T().Name(), role)
}

func (s *ObjectRepositorySuite) objectPrefix() string {
	s.T().Helper()

	return s.T().Name() + "/"
}

func (s *ObjectRepositorySuite) mustRecreateBucket(id, name, ownerID string) {
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

func (s *ObjectRepositorySuite) mustRecreateObject(id, bucketID, key string, currentVersionID *string, status string) {
	s.T().Helper()

	s.MustExec(`
		DELETE FROM objects
		WHERE id = $1 OR (bucket_id = $2 AND key = $3)
	`, id, bucketID, key)

	s.MustExec(`
		INSERT INTO objects (id, bucket_id, key, current_version_id, status)
		VALUES ($1, $2, $3, $4, $5)
	`, id, bucketID, key, currentVersionID, status)
}

func (s *ObjectRepositorySuite) mustSetCurrentVersion(objectID, versionID string) {
	s.T().Helper()

	s.MustExec(`
		UPDATE objects
		SET current_version_id = $2
		WHERE id = $1
	`, objectID, versionID)
}

func (s *ObjectRepositorySuite) mustRecreateVersion(
	id string,
	objectID string,
	blobID *string,
	sizeBytes int64,
	etag string,
	contentType string,
	versionNumber int64,
	kind string,
	state string,
) {
	s.T().Helper()

	s.MustExec(`
		DELETE FROM versions
		WHERE id = $1 OR (object_id = $2 AND version_number = $3)
	`, id, objectID, versionNumber)

	query := `
		INSERT INTO versions (
			id,
			object_id,
			blob_id,
			size_bytes,
			etag,
			content_type,
			version_number,
			kind,
			state,
			committed_at,
			aborted_at
		)
		VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9::version_state,
			CASE WHEN $9::version_state = 'committed'::version_state THEN now() ELSE NULL END,
			CASE WHEN $9::version_state = 'aborted'::version_state THEN now() ELSE NULL END
		)
	`

	s.MustExec(query, id, objectID, blobID, sizeBytes, etag, contentType, versionNumber, kind, state)
}

func strPtr(value string) *string {
	return &value
}

func (s *ObjectRepositorySuite) mustRecreateObjectWithoutCurrentVersion(id, key string) {
	s.T().Helper()

	s.mustRecreateBucket(s.bucketID(), s.bucketName(), s.ownerID())
	s.mustRecreateObject(id, s.bucketID(), key, nil, "active")
}

func (s *ObjectRepositorySuite) mustRecreateVersionBelongingToAnotherObject() string {
	s.T().Helper()

	targetObjectID := s.objectID(objectRoleOther)
	targetVersionID := s.versionID(objectRoleOther)
	targetBlobID := s.blobID(objectRoleOther)
	requestedObjectID := s.objectID(objectRoleActive)

	s.mustRecreateBucket(s.bucketID(), s.bucketName(), s.ownerID())
	s.mustRecreateObject(requestedObjectID, s.bucketID(), s.objectKey(objectRoleActive), nil, "active")
	s.mustRecreateObject(targetObjectID, s.bucketID(), s.objectKey(objectRoleOther), nil, "active")
	s.mustRecreateVersion(
		targetVersionID,
		targetObjectID,
		strPtr(targetBlobID),
		2048,
		`"etag-2"`,
		"image/jpeg",
		1,
		"blob",
		"committed",
	)

	return targetVersionID
}

func (s *ObjectRepositorySuite) mustRecreateListFixture() {
	s.T().Helper()

	activeObjectID := s.objectID(objectRoleActive)
	otherObjectID := s.objectID(objectRoleOther)
	deletedObjectID := s.objectID(objectRoleDeleted)
	activeVersionID := s.versionID(objectRoleActive)
	otherVersionID := s.versionID(objectRoleOther)
	deletedVersionID := s.versionID(objectRoleDeleted)

	s.mustRecreateBucket(s.bucketID(), s.bucketName(), s.ownerID())
	s.mustRecreateObject(activeObjectID, s.bucketID(), s.objectKey(objectRoleActive), nil, "active")
	s.mustRecreateObject(otherObjectID, s.bucketID(), s.objectKey(objectRoleOther), nil, "uploading")
	s.mustRecreateObject(deletedObjectID, s.bucketID(), s.objectKey(objectRoleDeleted), nil, "active")
	s.mustRecreateVersion(
		activeVersionID,
		activeObjectID,
		strPtr(s.blobID(objectRoleActive)),
		100,
		`"etag-1"`,
		"image/jpeg",
		1,
		"blob",
		"committed",
	)
	s.mustRecreateVersion(
		otherVersionID,
		otherObjectID,
		strPtr(s.blobID(objectRoleOther)),
		200,
		`"etag-2"`,
		"image/jpeg",
		1,
		"blob",
		"committed",
	)
	s.mustRecreateVersion(
		deletedVersionID,
		deletedObjectID,
		nil,
		0,
		"",
		"application/octet-stream",
		1,
		"delete_marker",
		"committed",
	)
	s.mustSetCurrentVersion(activeObjectID, activeVersionID)
	s.mustSetCurrentVersion(otherObjectID, otherVersionID)
	s.mustSetCurrentVersion(deletedObjectID, deletedVersionID)
}
