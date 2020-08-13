package cert

import (
	"crypto/tls"
	"fmt"
	"strings"
)

var DefaultMemProvider = NewMemProvider()

// MemProvider A simple in-memory certificate cache
type MemProvider struct {
	cache map[string]*tls.Certificate
}

// Create a MemProvider
func NewMemProvider() *MemProvider {
	return &MemProvider{
		cache: make(map[string]*tls.Certificate),
	}
}

// Get the certificate for the Host from the cache
func (m *MemProvider) Get(host string) (cert *tls.Certificate, err error) {
	var ok bool
	cert, ok = m.cache[strings.TrimSpace(host)]
	if !ok {
		err = fmt.Errorf("cert not exist")
	}
	return
}

// Set the Host certificate to the cache
func (m *MemProvider) Set(host string, cert *tls.Certificate) error {
	host = strings.TrimSpace(host)
	m.cache[host] = cert
	return nil
}
