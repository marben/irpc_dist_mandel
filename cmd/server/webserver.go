package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/coder/websocket"
)

// WebServer creates server serving files in ./static folder
// initializes websocket endpoint and returns net.Listener accepting websocket connections
func webServer(ctx context.Context, port int) (net.Listener, *http.Server) {
	l := NewWSListener(ctx, fmt.Sprintf(":%d/ws", port))
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", websocketHandler(l))
	mux.Handle("/", http.FileServer(http.Dir("./static")))

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("listening on http://localhost:%d", port)
	return l, srv
}

// websocketHandler handles the http ws endpoint
// if websocket is succesfully initialized it is passed to WebsocketListener so it can be accepted
func websocketHandler(l *WebsocketListener) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			OriginPatterns: []string{"*"}, // TODO: tighten in prod
		})
		if err != nil {
			log.Println(err)
			return
		}

		l.ch <- c
	}
}

// WebsocketListener implements net.Listener
// it's a wrapper around websocket.Conn
type WebsocketListener struct {
	ch     chan *websocket.Conn
	done   chan struct{}
	ctx    context.Context
	cancel context.CancelFunc
	addr   wsAddr
}

func NewWSListener(ctx context.Context, addr string) *WebsocketListener {
	ctx, cancel := context.WithCancel(ctx)
	return &WebsocketListener{
		ch:     make(chan *websocket.Conn),
		done:   make(chan struct{}),
		ctx:    ctx,
		cancel: cancel,
		addr:   wsAddr{addr: addr},
	}
}

func (l *WebsocketListener) Accept() (net.Conn, error) {
	select {
	case c := <-l.ch:
		return websocket.NetConn(l.ctx, c, websocket.MessageBinary), nil
	case <-l.ctx.Done():
		return nil, context.Cause(l.ctx)
	case <-l.done:
		return nil, net.ErrClosed
	}
}

func (l *WebsocketListener) Addr() net.Addr {
	return l.addr
}

func (l *WebsocketListener) Close() error {
	l.cancel()
	return nil
}

// wsAddrs implements net.Addr
type wsAddr struct {
	addr string
}

func (a wsAddr) Network() string {
	return "ws"
}

func (a wsAddr) String() string {
	return a.addr
}
