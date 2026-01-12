package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/coder/websocket"
)

func webServer(ctx context.Context) (net.Listener, *http.Server) {
	l := NewWSListener(ctx, ":8080/ws")
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", wsHandler(l))
	mux.Handle("/", http.FileServer(http.Dir("./static")))

	srv := &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Println("listening on http://localhost:8080")
	return l, srv
}

// WSListener is adapter from wsHandler to net.Listener()
type WSListener struct {
	ch     chan *websocket.Conn
	done   chan struct{}
	ctx    context.Context
	cancel context.CancelFunc
	addr   wsAddr
}

func NewWSListener(ctx context.Context, addr string) *WSListener {
	ctx, cancel := context.WithCancel(ctx)
	return &WSListener{
		ch:     make(chan *websocket.Conn),
		done:   make(chan struct{}),
		ctx:    ctx,
		cancel: cancel,
		addr:   wsAddr{addr: addr},
	}
}

func (l *WSListener) Accept() (net.Conn, error) {
	select {
	case c := <-l.ch:
		log.Printf("obtained ws conn: %v", c)
		return websocket.NetConn(l.ctx, c, websocket.MessageBinary), nil
	case <-l.ctx.Done():
		return nil, context.Cause(l.ctx)
	case <-l.done:
		return nil, net.ErrClosed
	}
}

type wsAddr struct {
	addr string
}

func (a wsAddr) Network() string {
	return "ws"
}

func (a wsAddr) String() string {
	return a.addr
}

func (l *WSListener) Addr() net.Addr {
	return l.addr
}

func (l *WSListener) Close() error {
	l.cancel()
	return nil
}

func wsHandler(l *WSListener) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			OriginPatterns: []string{"*"}, // tighten in prod
		})
		if err != nil {
			log.Println(err)
			return
		}

		l.ch <- c
	}
}
