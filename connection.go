package main

import (
	"io"
	"sync/atomic"
	"time"
)

const (
	KeepAlivePeriod      = 5 * time.Second
	FriendsRefreshPeriod = 30 * time.Minute
)

var (
	KeepAliveData = []byte("\r\n")
)

type Connection struct {
	w      io.Writer
	closed int32

	wait chan struct{}

	data chan []byte
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
		case <-time.After(KeepAlivePeriod):
			_, err = c.w.Write(KeepAliveData)

		case d := <-c.data:
			_, err = c.w.Write(d)
		}

		if err != nil && err != io.EOF {
			atomic.StoreInt32(&c.closed, 1)
			break
		}
	}
	// 채널 비우기
	close(c.data)
	close(c.wait)
}
