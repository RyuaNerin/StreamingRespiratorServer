package main

import (
	"context"
	"io"
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
	w      io.Writer
	closed int32

	ctx context.Context

	done chan struct{}

	data chan []byte
}

func newConnection(w io.Writer, ctx context.Context) *Connection {
	return &Connection{
		w:    w,
		ctx:  ctx,
		done: make(chan struct{}),
		data: make(chan []byte),
	}
}

func (c *Connection) Send(data []byte) {
	if atomic.LoadInt32(&c.closed) == 1 {
		return
	}

	c.data <- data
}

func (c *Connection) Broadcaster() {
	var err error

	for atomic.LoadInt32(&c.closed) == 0 {
		select {
		case <-c.ctx.Done():
			break

		case <-time.After(KeepAlivePeriod):
			_, err = c.w.Write(KeepAliveData)

		case d := <-c.data:
			_, err = c.w.Write(d)
		}

		if err != nil && err != io.EOF {
			break
		}
	}
	atomic.StoreInt32(&c.closed, 1)

	// 채널 비우기
	close(c.data)
	close(c.done)
}
