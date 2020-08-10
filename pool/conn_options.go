package pool

import "time"

var DefaultConnOptions = &ConnOptions{
	IdleMaxCap: 20,
	Timeout:    time.Minute,
}

type ConnOptions struct {
	IdleMaxCap int
	Timeout    time.Duration
}
