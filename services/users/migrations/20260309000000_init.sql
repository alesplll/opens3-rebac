-- +goose Up

CREATE TABLE users (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name       text NOT NULL,
    email      text NOT NULL,
    password   text NOT NULL,
    created_at timestamp without time zone NOT NULL DEFAULT now(),
    updated_at timestamp without time zone NOT NULL DEFAULT now(),
    CONSTRAINT unique_email UNIQUE (email)
);

CREATE TABLE password_change_logs (
    id         serial PRIMARY KEY,
    user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    changed_at timestamp without time zone NOT NULL DEFAULT now(),
    ip_address text
);

-- +goose Down

DROP TABLE IF EXISTS password_change_logs;
DROP TABLE IF EXISTS users;
