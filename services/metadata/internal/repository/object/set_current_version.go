package object

import (
	"context"

	sq "github.com/Masterminds/squirrel"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/client/db"
)

// SetCurrentVersion updates current_version_id pointer on the object.
// Called as the last step in CreateObjectVersion transaction.
func (r *repo) SetCurrentVersion(ctx context.Context, objectID, versionID string) error {
	builder := sq.Update(objectsTable).
		PlaceholderFormat(sq.Dollar).
		Set("current_version_id", versionID).
		Where(sq.Eq{"id": objectID})

	query, args, err := builder.ToSql()
	if err != nil {
		return err
	}

	q := db.Query{
		Name:     "object_repository:SetCurrentVersion",
		QueryRaw: query,
	}

	_, err = r.db.DB().ExecContext(ctx, q, args...)
	return err
}
