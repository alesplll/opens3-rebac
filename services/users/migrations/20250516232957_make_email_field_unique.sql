-- +goose Up
DELETE FROM users 
WHERE id NOT IN (
  SELECT MIN(id)
  FROM users 
  GROUP BY email
);

ALTER TABLE users ADD CONSTRAINT unique_email UNIQUE (email);

-- +goose Down
ALTER TABLE users DROP CONSTRAINT unique_email;
