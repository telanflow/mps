package mps

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestNewTunnelHandler(t *testing.T) {
	srv := NewTestServer()
	defer srv.Close()

	tunnel := NewTunnelHandler()
	//tunnel.Transport().Proxy = func(r *http.Request) (*url.URL, error) {
	//	return url.Parse("http://127.0.0.1:7890")
	//}
	tunnelSrv := httptest.NewServer(tunnel)
	defer tunnelSrv.Close()

	resp, err := HttpGet(srv.URL, func(r *http.Request) (*url.URL, error) {
		return url.Parse(tunnelSrv.URL)
	})
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	asserts := assert.New(t)
	asserts.Equal(resp.StatusCode, 200)
}
