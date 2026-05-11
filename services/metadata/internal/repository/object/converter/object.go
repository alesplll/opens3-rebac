package converter

import (
	domainModel "github.com/alesplll/opens3-rebac/services/metadata/internal/model"
	repoModel "github.com/alesplll/opens3-rebac/services/metadata/internal/repository/object/model"
)

func MetaToDomain(m *repoModel.ObjectMeta) *domainModel.ObjectMeta {
	return &domainModel.ObjectMeta{
		ObjectID:     m.ObjectID,
		VersionID:    m.VersionID,
		BlobID:       m.BlobID,
		SizeBytes:    m.SizeBytes,
		Etag:         m.Etag,
		ContentType:  m.ContentType,
		LastModified: m.LastModified,
	}
}

func ListItemToDomain(item *repoModel.ObjectListItem) *domainModel.ObjectListItem {
	return &domainModel.ObjectListItem{
		ObjectID:     item.ObjectID,
		VersionID:    item.VersionID,
		Key:          item.Key,
		Etag:         item.Etag,
		SizeBytes:    item.SizeBytes,
		ContentType:  item.ContentType,
		LastModified: item.LastModified,
	}
}
