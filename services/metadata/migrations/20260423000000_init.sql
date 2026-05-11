-- +goose Up

CREATE TYPE version_state AS ENUM (
    'pending',
    'committed',
    'aborted'
);

CREATE TYPE version_kind AS ENUM (
    'blob',
    'delete_marker'
);

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
    pending_version_id UUID NULL,
    status             TEXT NOT NULL DEFAULT 'active',
    CONSTRAINT chk_objects_distinct_version_pointers
        CHECK (
            current_version_id IS NULL
            OR pending_version_id IS NULL
            OR current_version_id <> pending_version_id
        ),
    UNIQUE (bucket_id, key)
);

CREATE TABLE versions (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    object_id        UUID NOT NULL REFERENCES objects(id) ON DELETE CASCADE,
    blob_id          UUID NULL,
    size_bytes       BIGINT NOT NULL DEFAULT 0,
    etag             TEXT NOT NULL DEFAULT '',
    content_type     TEXT NOT NULL DEFAULT 'application/octet-stream',
    version_number   BIGINT NOT NULL,
    kind             version_kind NOT NULL DEFAULT 'blob',
    state            version_state NOT NULL DEFAULT 'pending',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    committed_at     TIMESTAMPTZ NULL,
    aborted_at       TIMESTAMPTZ NULL,
    UNIQUE (object_id, version_number),
    CONSTRAINT chk_versions_delete_marker_state
        CHECK (kind <> 'delete_marker' OR state = 'committed'),
    CONSTRAINT chk_versions_delete_marker_blob
        CHECK (kind <> 'delete_marker' OR blob_id IS NULL),
    CONSTRAINT chk_versions_committed_blob
        CHECK (state <> 'committed' OR kind = 'delete_marker' OR blob_id IS NOT NULL),
    CONSTRAINT chk_versions_committed_at_state
        CHECK (
            (state = 'committed' AND committed_at IS NOT NULL)
            OR
            (state <> 'committed' AND committed_at IS NULL)
        ),
    CONSTRAINT chk_versions_aborted_at_state
        CHECK (
            (state = 'aborted' AND aborted_at IS NOT NULL)
            OR
            (state <> 'aborted' AND aborted_at IS NULL)
        ),
    CONSTRAINT chk_versions_delete_marker_pending_forbidden
        CHECK (kind <> 'delete_marker' OR state <> 'pending'),
    CONSTRAINT chk_versions_commit_after_create
        CHECK (
            committed_at IS NULL
            OR committed_at >= created_at
        ),
    CONSTRAINT chk_versions_abort_after_create
        CHECK (
            aborted_at IS NULL
            OR aborted_at >= created_at
        )
);

ALTER TABLE objects
    ADD CONSTRAINT fk_objects_current_version
        FOREIGN KEY (current_version_id) REFERENCES versions(id)
        DEFERRABLE INITIALLY DEFERRED,
    ADD CONSTRAINT fk_objects_pending_version
        FOREIGN KEY (pending_version_id) REFERENCES versions(id)
        DEFERRABLE INITIALLY DEFERRED;

CREATE INDEX ON objects(bucket_id, key);
CREATE INDEX ON objects(bucket_id, key text_pattern_ops);
CREATE INDEX ON objects(bucket_id, status, key);
CREATE INDEX ON versions(object_id);
CREATE INDEX ON versions(object_id, state);
CREATE INDEX ON versions(object_id, kind);
CREATE UNIQUE INDEX ux_versions_one_pending_per_object
    ON versions(object_id)
    WHERE state = 'pending';

-- +goose Down

ALTER TABLE objects
    DROP CONSTRAINT IF EXISTS fk_objects_current_version,
    DROP CONSTRAINT IF EXISTS fk_objects_pending_version;

DROP INDEX IF EXISTS versions_object_id_state_idx;
DROP INDEX IF EXISTS versions_object_id_idx;
DROP INDEX IF EXISTS versions_object_id_kind_idx;
DROP INDEX IF EXISTS ux_versions_one_pending_per_object;
DROP INDEX IF EXISTS objects_bucket_id_status_key_idx;
DROP INDEX IF EXISTS objects_bucket_id_key_text_pattern_ops_idx;
DROP INDEX IF EXISTS objects_bucket_id_key_idx;
DROP TABLE IF EXISTS versions;
DROP TABLE IF EXISTS objects;
DROP TABLE IF EXISTS buckets;
DROP TYPE IF EXISTS version_kind;
DROP TYPE IF EXISTS version_state;
