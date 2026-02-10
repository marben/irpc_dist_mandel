package main

import (
	"io"
	"sync"
	"syscall/js"
)

// WebsocketReadWriteCloser provides io.ReadWriteCloser interface over a JavaScript WebSocket (js.Value).
type WebsocketReadWriteCloser struct {
	ws js.Value

	mu     sync.Mutex // needed because js onClose event can preempt Write() call
	closed bool

	readCh chan []byte

	openCh chan struct{} // closed when connected
	err    error

	// read buffer for partial reads
	buf []byte
}

func NewWebsocketReadWriteCloser(ws js.Value) *WebsocketReadWriteCloser {
	c := &WebsocketReadWriteCloser{
		ws:     ws,
		readCh: make(chan []byte, 8),
		openCh: make(chan struct{}),
	}

	ws.Set("binaryType", "arraybuffer")

	ws.Set("onopen", js.FuncOf(func(js.Value, []js.Value) any {
		close(c.openCh)
		return nil
	}))

	ws.Set("onerror", js.FuncOf(func(js.Value, []js.Value) any {
		c.mu.Lock()
		c.err = io.ErrUnexpectedEOF
		c.mu.Unlock()
		close(c.openCh)
		return nil
	}))

	ws.Set("onmessage", js.FuncOf(func(this js.Value, args []js.Value) any {
		data := args[0].Get("data")

		jsDataToBytes(data, func(b []byte) {
			c.readCh <- b
		})

		return nil
	}))

	ws.Set("onclose", js.FuncOf(func(js.Value, []js.Value) any {
		logScreenf("ws onClose received")
		c.mu.Lock()
		c.closed = true
		close(c.readCh)
		c.mu.Unlock()
		return nil
	}))

	return c
}

func (c *WebsocketReadWriteCloser) Read(p []byte) (int, error) {
	// First, drain existing buffer
	if len(c.buf) == 0 {
		// No buffered data -> wait for next message
		msg, ok := <-c.readCh
		if !ok {
			return 0, io.EOF
		}
		c.buf = msg
	}

	n := copy(p, c.buf)
	c.buf = c.buf[n:]

	return n, nil
}

func (c *WebsocketReadWriteCloser) Write(p []byte) (int, error) {
	if err := c.waitOpen(); err != nil {
		return 0, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return 0, io.ErrClosedPipe
	}

	u8 := js.Global().Get("Uint8Array").New(len(p))
	js.CopyBytesToJS(u8, p)

	c.ws.Call("send", u8)
	return len(p), nil
}

func (c *WebsocketReadWriteCloser) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}

	c.closed = true

	select {
	case <-c.openCh:
	default:
		close(c.openCh)
	}

	close(c.readCh)
	c.mu.Unlock()

	c.ws.Call("close")
	return nil
}

func (c *WebsocketReadWriteCloser) waitOpen() error {
	<-c.openCh

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.err != nil {
		return c.err
	}
	if c.closed {
		return io.ErrClosedPipe
	}
	return nil
}

func jsDataToBytes(data js.Value, deliver func([]byte)) {
	// Uint8Array / Uint8ClampedArray
	if data.InstanceOf(js.Global().Get("Uint8Array")) ||
		data.InstanceOf(js.Global().Get("Uint8ClampedArray")) {

		b := make([]byte, data.Get("byteLength").Int())
		js.CopyBytesToGo(b, data)
		deliver(b)
		return
	}

	// ArrayBuffer
	if data.InstanceOf(js.Global().Get("ArrayBuffer")) {
		u8 := js.Global().Get("Uint8Array").New(data)
		b := make([]byte, u8.Get("byteLength").Int())
		js.CopyBytesToGo(b, u8)
		deliver(b)
		return
	}

	// Blob → async
	if data.InstanceOf(js.Global().Get("Blob")) {
		promise := data.Call("arrayBuffer")
		then := js.FuncOf(func(this js.Value, args []js.Value) any {
			buf := args[0]
			u8 := js.Global().Get("Uint8Array").New(buf)
			b := make([]byte, u8.Get("byteLength").Int())
			js.CopyBytesToGo(b, u8)
			deliver(b)
			return nil
		})
		promise.Call("then", then)
		return
	}

	panic("unsupported JS binary type")
}
