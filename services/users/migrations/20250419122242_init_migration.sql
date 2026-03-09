-- +goose Up
create type user_role as enum ('ADMIN', 'USER');
create table users (
  id serial primary key,
  name text not null,
  email text not null,
  password text not null,
  role user_role not null default 'USER',
  created_at timestamp without time zone not null default now(),
  updated_at timestamp without time zone not null default now()
);

-- +goose Down
drop table users;
drop type user_role;
