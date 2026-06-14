package internal

import (
	"time"
)

type ApiEndpoint struct {
	Name string
	Url  string
}

type Result struct {
	Name     string
	Body     []byte
	Duration time.Duration
	Error    error
}
