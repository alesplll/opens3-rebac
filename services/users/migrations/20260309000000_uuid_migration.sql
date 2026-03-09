-- +goose Up
ALTER TABLE password_change_logs DROP CONSTRAINT IF EXISTS password_change_logs_user_id_fkey;
ALTER TABLE users ALTER COLUMN id DROP DEFAULT;
ALTER TABLE users ALTER COLUMN id SET DATA TYPE uuid USING gen_random_uuid();
ALTER TABLE users ALTER COLUMN id SET DEFAULT gen_random_uuid();
ALTER TABLE password_change_logs ALTER COLUMN user_id SET DATA TYPE uuid USING user_id::uuid;
ALTER TABLE password_change_logs ADD CONSTRAINT password_change_logs_user_id_fkey
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

-- +goose Down
-- Note: downward migration from UUID to serial is destructive
ALTER TABLE password_change_logs DROP CONSTRAINT IF EXISTS password_change_logs_user_id_fkey;
ALTER TABLE users ALTER COLUMN id SET DATA TYPE bigint USING 0;
ALTER TABLE users ALTER COLUMN id SET DEFAULT nextval('users_id_seq');
ALTER TABLE password_change_logs ALTER COLUMN user_id SET DATA TYPE integer USING 0;
ALTER TABLE password_change_logs ADD CONSTRAINT password_change_logs_user_id_fkey
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
