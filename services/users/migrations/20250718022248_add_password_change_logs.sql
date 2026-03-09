-- +goose Up
create table password_change_logs (
  id serial primary key,
  user_id integer not null references users(id) on delete CASCADE,
  changed_at timestamp without time zone not null default now(),
  ip_address text
);

-- +goose Down
drop table if exists password_change_logs;
