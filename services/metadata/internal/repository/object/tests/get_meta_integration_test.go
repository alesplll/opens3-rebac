//go:build integration

package tests

import (
	domainerrors "github.com/alesplll/opens3-rebac/services/metadata/internal/errors/domain_errors"
)

func (s *ObjectRepositorySuite) TestGetMeta_VersionMustBelongToRequestedObject() {
	versionID := s.mustRecreateVersionBelongingToAnotherObject()

	got, err := s.repo.GetMeta(s.Context(), s.bucketName(), s.objectKey(objectRoleActive), versionID)

	s.Require().Nil(got)
	s.Require().ErrorIs(err, domainerrors.ErrObjectNotFound)
	s.Require().Equal(int64(2), s.MustCount("objects"))
	s.Require().Equal(int64(1), s.MustCount("versions"))
}
