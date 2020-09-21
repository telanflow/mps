package mps

import (
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
)

func TestNewReverseHandler(t *testing.T) {
	srv := NewTestServer()
	defer srv.Close()

	reverseHandler := NewReverseHandler()
	proxySrv := httptest.NewServer(reverseHandler)
	defer proxySrv.Close()

	resp, err := HttpGet(srv.URL, func(r *http.Request) (*url.URL, error) {
		return url.Parse(proxySrv.URL)
	})
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	bodySize := len(body)
	contentLength, _ := strconv.Atoi(resp.Header.Get("Content-Length"))

	asserts := assert.New(t)
	asserts.Equal(resp.StatusCode, 200, "statusCode should be equal 200")
	asserts.Equal(bodySize, contentLength, "Content-Length should be equal " + strconv.Itoa(bodySize))
	asserts.Equal(int64(bodySize), resp.ContentLength)
}
