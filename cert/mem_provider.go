package cert

import (
	"crypto/tls"
	"fmt"
	"strings"
)

type MemProvider struct {
	cache map[string]*tls.Certificate
}

func NewMemProvider() *MemProvider {
	return &MemProvider{
		cache: make(map[string]*tls.Certificate),
	}
}

func (m *MemProvider) Get(host string) (cert *tls.Certificate, err error) {
	var ok bool
	cert, ok = m.cache[strings.TrimSpace(host)]
	if !ok {
		err = fmt.Errorf("cert not exist")
	}
	return
}

func (m *MemProvider) Set(host string, cert *tls.Certificate) error {
	host = strings.TrimSpace(host)
	m.cache[host] = cert
	return nil
}
