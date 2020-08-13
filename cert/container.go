package cert

import "crypto/tls"

// certificate storage Container
type Container interface {

	// Get the certificate for host
	Get(host string) (*tls.Certificate, error)

	// Set the certificate for host
	Set(host string, cert *tls.Certificate) error
}
