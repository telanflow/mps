package mps

import (
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{}

// create a test websocket server
func newTestWebsocketServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		c, err := upgrader.Upgrade(rw, req, nil)
		if err != nil {
			return
		}
		defer c.Close()
		for {
			mt, message, err := c.ReadMessage()
			if err != nil {
				break
			}
			err = c.WriteMessage(mt, message)
			if err != nil {
				break
			}
		}
	}))
}

func TestNewWebsocketHandler(t *testing.T) {
	// create endPoint websocket server
	srv := newTestWebsocketServer()
	defer srv.Close()

	// Convert http://127.0.0.1 to ws://127.0.0.1
	endPoint := "ws" + strings.TrimPrefix(srv.URL, "http")
	log.Printf("endPoint: %s", endPoint)

	// create a proxy websocket server
	wsHandler := NewWebsocketHandler()
	wsHandler.Transport().Proxy = func(request *http.Request) (*url.URL, error) {
		return url.Parse(endPoint)
	}
	proxySrv := httptest.NewServer(wsHandler)
	defer proxySrv.Close()

	proxyWs := "ws" + strings.TrimPrefix(proxySrv.URL, "http")
	log.Printf("proxy: %s", proxyWs)

	// Connect to the proxy websocket server
	client, _, err := websocket.DefaultDialer.Dial(proxyWs, nil)
	if err != nil {
		t.Fatalf("%v", err)
	}
	defer client.Close()

	// Send message to server, read response and check to see if it's what we expect.
	for i := 0; i < 5; i++ {
		if err := client.WriteMessage(websocket.TextMessage, []byte("hello")); err != nil {
			t.Fatalf("send fail: %v", err)
		}

		_, p, err := client.ReadMessage()
		if err != nil {
			t.Fatalf("read fail: %v", err)
		}

		log.Printf("recv: %s", string(p))
		if string(p) != "hello" {
			t.Fatalf("bad message")
		}
	}
}
