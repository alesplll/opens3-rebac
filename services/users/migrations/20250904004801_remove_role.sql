-- +goose Up
alter table users drop column role;
drop type user_role;

-- +goose Down
create type user_role as enum ('ADMIN', 'USER');
alter table users add column role user_role not null default 'USER';
