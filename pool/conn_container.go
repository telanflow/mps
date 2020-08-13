package pool

import "net"

// ConnContainer connection pool interface
type ConnContainer interface {
	// Get returned a idle net.Conn
	Get(addr string) (net.Conn, error)

	// Put place a idle net.Conn into the pool
	Put(conn net.Conn) error

	// Release connection pool
	Release() error
}
