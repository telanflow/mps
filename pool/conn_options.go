package pool

import "time"

var DefaultConnOptions = &ConnOptions{
	IdleMaxCap: 30,
	Timeout:    90 * time.Second,
}

// ConnOptions is ConnProvider options
type ConnOptions struct {
	// IdleMaxCap is max connection capacity for a single net.Addr
	IdleMaxCap int

	// Timeout specifies how long the connection will timeout
	Timeout time.Duration
}
