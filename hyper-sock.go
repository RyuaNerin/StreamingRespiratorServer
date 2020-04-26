package main

import (
	"crypto/tls"
	"io"
	"net"
	"sync"
	"time"

	"github.com/getsentry/sentry-go"
)

type hyperListener struct {
	l   net.Listener
	cfg *tls.Config
}

func newHyperListner(l net.Listener, tlsConfig *tls.Config) *hyperListener {
	return &hyperListener{
		l:   l,
		cfg: tlsConfig,
	}
}

func (hl *hyperListener) Accept() (net.Conn, error) {
	c, err := hl.l.Accept()
	if err != nil {
		logger.Printf("%+v\n", err)
		sentry.CaptureException(err.(error))
		return nil, err
	}
	return newHyperConn(c, hl.cfg), nil
}
func (hl *hyperListener) Close() error {
	return hl.Close()
}
func (hl *hyperListener) Addr() net.Addr {
	return hl.Addr()
}

type hyperConn struct {
	conn net.Conn
	cfg  *tls.Config

	handshakeLock sync.Mutex
	handshaked    bool
}

func newHyperConn(c net.Conn, tlsConfig *tls.Config) *hyperConn {
	return &hyperConn{
		conn: c,
		cfg:  tlsConfig,
	}
}

func (hc *hyperConn) Read(b []byte) (n int, err error) {
	if err := hc.handshake(); err != nil {
		logger.Printf("%+v\n", err)
		sentry.CaptureException(err.(error))
		return 0, err
	}

	return hc.conn.Read(b)
}
func (hc *hyperConn) handshake() error {
	hc.handshakeLock.Lock()
	defer hc.handshakeLock.Unlock()

	if hc.handshaked {
		return nil
	}
	hc.handshaked = true

	buff := make([]byte, 6)
	r, err := hc.conn.Read(buff)
	if r != 6 {
		return io.ErrUnexpectedEOF
	}
	if err != nil {
		logger.Printf("%+v\n", err)
		sentry.CaptureException(err.(error))
		return err
	}

	ci := new(hyperConnInner)
	ci.Conn = &hyperConnBuffered{
		inner: ci,
		c:     hc.conn,
		buff:  buff,
	}

	hc.conn = ci

	if buff[0] == 0x16 && buff[1] == 0x03 && buff[5] == 0x01 {
		hc.conn = tls.Server(
			hc.conn,
			hc.cfg,
		)
	}
	return nil
}

func (hc *hyperConn) Write(b []byte) (n int, err error) {
	return hc.conn.Write(b)
}
func (hc *hyperConn) Close() error {
	return hc.conn.Close()
}
func (hc *hyperConn) LocalAddr() net.Addr {
	return hc.conn.LocalAddr()
}
func (hc *hyperConn) RemoteAddr() net.Addr {
	return hc.conn.RemoteAddr()
}
func (hc *hyperConn) SetDeadline(t time.Time) error {
	return hc.conn.SetDeadline(t)
}
func (hc *hyperConn) SetReadDeadline(t time.Time) error {
	return hc.conn.SetReadDeadline(t)
}
func (hc *hyperConn) SetWriteDeadline(t time.Time) error {
	return hc.conn.SetWriteDeadline(t)
}

type hyperConnInner struct {
	net.Conn
}

type hyperConnBuffered struct {
	inner *hyperConnInner
	c     net.Conn
	buff  []byte
}

func (hc *hyperConnBuffered) Read(b []byte) (n int, err error) {
	n = copy(b, hc.buff)
	if n > 0 {
		hc.buff = hc.buff[n:]

		if len(hc.buff) == 0 {
			hc.inner.Conn = hc.c
		}
	}

	if n < len(b) {
		var nn int
		nn, err = hc.c.Read(b[n:])
		n += nn
	}

	return
}
func (hc *hyperConnBuffered) Write(b []byte) (n int, err error) {
	return hc.c.Write(b)
}
func (hc *hyperConnBuffered) Close() error {
	return hc.c.Close()
}
func (hc *hyperConnBuffered) LocalAddr() net.Addr {
	return hc.c.LocalAddr()
}
func (hc *hyperConnBuffered) RemoteAddr() net.Addr {
	return hc.c.RemoteAddr()
}
func (hc *hyperConnBuffered) SetDeadline(t time.Time) error {
	return hc.c.SetDeadline(t)
}
func (hc *hyperConnBuffered) SetReadDeadline(t time.Time) error {
	return hc.c.SetReadDeadline(t)
}
func (hc *hyperConnBuffered) SetWriteDeadline(t time.Time) error {
	return hc.c.SetWriteDeadline(t)
}
