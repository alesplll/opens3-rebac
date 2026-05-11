package model

import "time"

type Bucket struct {
	ID        string
	Name      string
	OwnerID   string
	CreatedAt time.Time
}
