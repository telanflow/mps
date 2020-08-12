package middleware

import (
	"bytes"
	"encoding/base64"
	"github.com/telanflow/mps"
	"io/ioutil"
	"net/http"
	"strings"
)

// proxy Authorization header
const proxyAuthorization = "Proxy-Authorization"

// BasicAuth returns a HTTP Basic Authentication middleware for requests
// You probably want to use mps.BasicAuth(proxy) to enable authentication for all proxy activities
func BasicAuth(realm string, fn func(username, password string) bool) mps.MiddlewareFunc {
	return func(req *http.Request, ctx *mps.Context) (*http.Response, error) {
		auth := req.Header.Get(proxyAuthorization)
		if auth == "" {
			return BasicUnauthorized(req, realm), nil
		}
		// parses an Basic Authentication string.
		usr, pwd, ok := parseBasicAuth(auth)
		if !ok {
			return BasicUnauthorized(req, realm), nil
		}
		if !fn(usr, pwd) {
			return BasicUnauthorized(req, realm), nil
		}
		// Authorization passed
		return ctx.Next(req)
	}
}

// SetBasicAuth sets the request's Authorization header to use HTTP
// Basic Authentication with the provided username and password.
//
// With HTTP Basic Authentication the provided username and password
// are not encrypted.
//
// Some protocols may impose additional requirements on pre-escaping the
// username and password. For instance, when used with OAuth2, both arguments
// must be URL encoded first with url.QueryEscape.
func SetBasicAuth(req *http.Request, username, password string) {
	req.Header.Set(proxyAuthorization, "Basic "+basicAuth(username, password))
}

// See 2 (end of page 4) https://www.ietf.org/rfc/rfc2617.txt
// "To receive authorization, the client sends the userid and password,
// separated by a single colon (":") character, within a base64
// encoded string in the credentials."
// It is not meant to be urlencoded.
func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

// parseBasicAuth parses an HTTP Basic Authentication string.
// "Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==" returns ("Aladdin", "open sesame", true).
func parseBasicAuth(auth string) (username, password string, ok bool) {
	const prefix = "Basic "
	// Case insensitive prefix match. See Issue 22736.
	if len(auth) < len(prefix) || !strings.EqualFold(auth[:len(prefix)], prefix) {
		return
	}
	c, err := base64.StdEncoding.DecodeString(auth[len(prefix):])
	if err != nil {
		return
	}
	cs := string(c)
	s := strings.IndexByte(cs, ':')
	if s < 0 {
		return
	}
	return cs[:s], cs[s+1:], true
}

func BasicUnauthorized(req *http.Request, realm string) *http.Response {
	const unauthorizedMsg = "407 Proxy Authentication Required"
	// verify realm is well formed
	return &http.Response{
		StatusCode: 407,
		ProtoMajor: 1,
		ProtoMinor: 1,
		Request:    req,
		Header: http.Header{
			"Proxy-Authenticate": []string{"Basic realm=" + realm},
			"Proxy-Connection":   []string{"close"},
		},
		Body:          ioutil.NopCloser(bytes.NewBuffer([]byte(unauthorizedMsg))),
		ContentLength: int64(len(unauthorizedMsg)),
	}
}
