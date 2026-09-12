//go:build integration

package tests

import (
	"fmt"

	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/client/db"
)

func (s *ObjectRepositorySuite) TestInsertVersionAssignsMonotonicVersionNumbers() {
	objectID := s.objectID(objectRoleActive)
	s.mustRecreateObjectWithoutCurrentVersion(objectID, s.objectKey(objectRoleActive))

	for i := 1; i <= 2; i++ {
		blobID := s.FixtureUUID(s.T().Name(), "insert-version-blob", fmt.Sprint(i))
		_, _, err := s.repo.InsertVersion(
			s.Context(),
			objectID,
			blobID,
			int64(i*100),
			fmt.Sprintf("etag-%d", i),
			"application/octet-stream",
		)
		s.Require().NoError(err)
	}

	rows, err := s.client.DB().QueryContext(
		s.Context(),
		db.Query{
			Name:     "object_repository_tests:version_numbers",
			QueryRaw: "SELECT version_number FROM versions WHERE object_id = $1 ORDER BY version_number",
		},
		objectID,
	)
	s.Require().NoError(err)
	defer rows.Close()

	var got []int64
	for rows.Next() {
		var number int64
		s.Require().NoError(rows.Scan(&number))
		got = append(got, number)
	}
	s.Require().NoError(rows.Err())
	s.Equal([]int64{1, 2}, got)
}
