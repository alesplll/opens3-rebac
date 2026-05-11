package config

import "time"

type PGConfig interface {
	DSN() string
	Timeout() time.Duration
	NeedLog() bool
}
