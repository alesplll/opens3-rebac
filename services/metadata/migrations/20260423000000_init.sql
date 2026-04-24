-- +goose Up

CREATE TABLE buckets (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT UNIQUE NOT NULL,
    owner_id   UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE objects (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bucket_id          UUID NOT NULL REFERENCES buckets(id) ON DELETE CASCADE,
    key                TEXT NOT NULL,
    current_version_id UUID NULL,
    UNIQUE (bucket_id, key)
);

CREATE TABLE versions (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    object_id    UUID NOT NULL REFERENCES objects(id) ON DELETE CASCADE,
    blob_id      UUID NOT NULL,
    size_bytes   BIGINT NOT NULL,
    etag         TEXT NOT NULL,
    content_type TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    is_deleted   BOOLEAN NOT NULL DEFAULT false
);

CREATE INDEX ON objects(bucket_id, key);
CREATE INDEX ON versions(object_id);
CREATE INDEX ON objects(bucket_id, key text_pattern_ops);

-- +goose Down

DROP INDEX IF EXISTS objects_bucket_id_key_text_pattern_ops_idx;
DROP INDEX IF EXISTS objects_bucket_id_key_idx;
DROP INDEX IF EXISTS versions_object_id_idx;
DROP TABLE IF EXISTS versions;
DROP TABLE IF EXISTS objects;
DROP TABLE IF EXISTS buckets;
