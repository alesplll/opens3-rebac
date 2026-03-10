package model

import "time"

type UserInfo struct {
	Name  string `db:"name"`
	Email string `db:"email"`
}

type User struct {
	Id        string    `db:"id"`
	UserInfo  UserInfo  `db:""`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}
