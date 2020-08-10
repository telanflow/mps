package pool

import "net"

type ConnContainer interface {
	Get(addr string) (net.Conn, error)
	Put(conn net.Conn) error
}
