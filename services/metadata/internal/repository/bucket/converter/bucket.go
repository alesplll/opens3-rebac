package converter

import (
	domainModel "github.com/alesplll/opens3-rebac/services/metadata/internal/model"
	repoModel "github.com/alesplll/opens3-rebac/services/metadata/internal/repository/bucket/model"
)

func FromRepoToDomain(b *repoModel.Bucket) *domainModel.Bucket {
	return &domainModel.Bucket{
		ID:        b.ID,
		Name:      b.Name,
		OwnerID:   b.OwnerID,
		CreatedAt: b.CreatedAt,
	}
}
