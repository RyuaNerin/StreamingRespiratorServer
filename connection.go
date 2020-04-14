package main

import (
	"context"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

const (
	KeepAlivePeriod = 5 * time.Second
)

var (
	KeepAliveData = []byte("\r\n")
)

type Connection struct {
	w        http.ResponseWriter
	wFlusher http.Flusher

	chanClosedRequest <-chan struct{} // Request 닫힘
	chanClosedWriter  <-chan bool     // ResponseWriter 닫힘

	chanClosed chan struct{} // 이 커넥션 닫힘

	closed int32

	writeLock    sync.Mutex
	tmrKeepAlive *time.Timer
}

func newConnection(w http.ResponseWriter, ctx context.Context) *Connection {
	c := &Connection{
		w:                 w,
		wFlusher:          w.(http.Flusher),
		chanClosedRequest: ctx.Done(),
		chanClosed:        make(chan struct{}),
		tmrKeepAlive:      time.NewTimer(0),
	}

	if cn, ok := w.(http.CloseNotifier); ok {
		c.chanClosedWriter = cn.CloseNotify()
	}

	c.w.Header().Set("Transfer-Encoding", "chunked")
	c.w.Header().Set("Content-type", "application/json; charset=utf-8")
	c.w.Header().Set("Connection", "close")
	c.w.WriteHeader(http.StatusOK)

	go c.keepAlive()
	return c
}

func (c *Connection) Send(data []byte) {
	c.sendInner(data, true)
}
func (c *Connection) sendInner(data []byte, resetKeepAlive bool) {
	if atomic.LoadInt32(&c.closed) == 1 {
		return
	}
	c.writeLock.Lock()
	defer c.writeLock.Unlock()

	select {
	case <-c.chanClosedRequest:
	case <-c.chanClosedWriter:
	default:
		n, err := c.w.Write(data)
		if n == len(data) && (err == nil || err == io.EOF) {
			if c.wFlusher != nil {
				c.wFlusher.Flush()
			}
			return
		}
	}

	// 채널 비우기
	atomic.StoreInt32(&c.closed, 1)
	if resetKeepAlive {
		c.tmrKeepAlive.Reset(KeepAlivePeriod)
	}
}

// keep-alive 패킷 보내는 역할
func (c *Connection) keepAlive() {
	for atomic.LoadInt32(&c.closed) == 0 {
		<-c.tmrKeepAlive.C
		c.sendInner(KeepAliveData, false)
	}
}
