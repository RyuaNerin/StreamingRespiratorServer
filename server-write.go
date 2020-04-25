package main

import (
	"bufio"
	"context"
	"io"
	"net/http"
	"runtime"
	"strconv"
	"sync"
)

const (
	CopyBufferSize = 4 * 1024 // 4 KiB
)

var (
	CopyBuffer = sync.Pool{
		New: func() interface{} {
			return make([]byte, CopyBufferSize)
		},
	}
)

func (s *streamingRespiratorServer) writeBytes(w http.ResponseWriter, statusCode int, responseBody []byte) error {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Length", strconv.Itoa(len(responseBody)))

	w.WriteHeader(statusCode)
	_, err := w.Write(responseBody)
	return err
}

func (s *streamingRespiratorServer) writeResponse(w http.ResponseWriter, resp *http.Response, body io.Reader, contentLength int) error {
	wh := w.Header()

	s.copyHeader(wh, resp.Header)
	resp.Header.Del("Content-Length")
	if contentLength != 0 {
		w.Header().Set("Content-Length", strconv.Itoa(contentLength))
	}

	if resp.Header != nil {
		resp.Header.Del("Content-Encoding")
	}

	if body == nil {
		body = resp.Body
	}

	w.WriteHeader(resp.StatusCode)
	_, err := io.Copy(w, body)
	return err
}

func (s *streamingRespiratorServer) copyHeader(dst http.Header, src http.Header) {
	for k, vr := range src {
		switch k {
		case "Content-Encoding":

		default:
			dst.Del(k)

			for _, v := range vr {
				dst.Add(k, v)
			}
		}
	}
}

func (s *streamingRespiratorServer) copy(client io.ReadWriter, clientReader *bufio.Reader, remote io.ReadWriter, ctx context.Context) {
	ctx, ctxCancel := context.WithCancel(ctx)

	clientReaderWithContext := readerWithContext{
		br:  clientReader,
		ctx: ctx,
	}
	remoteReaderWithContext := readerWithContext{
		br:  bufio.NewReader(remote),
		ctx: ctx,
	}

	done := make(chan struct{}, 1)
	go s.copyOneway(remote, &clientReaderWithContext, done, ctxCancel, "client -> remote")
	s.copyOneway(client, &remoteReaderWithContext, nil, ctxCancel, "remote -> client")
	<-done
}
func (s *streamingRespiratorServer) copyOneway(dst io.Writer, src io.Reader, ch chan struct{}, cancel context.CancelFunc, desc string) {
	defer cancel()

	buf := CopyBuffer.Get().([]byte)
	defer CopyBuffer.Put(buf)

	io.CopyBuffer(dst, src, buf)

	logger.Println(desc)

	if ch != nil {
		ch <- struct{}{}
	}
}

type readerWithContext struct {
	br  *bufio.Reader
	ctx context.Context
}

func (r *readerWithContext) Read(b []byte) (n int, err error) {
	ch := make(chan error, 1)
	go func() {
		_, err := r.br.Peek(1)
		ch <- err
		close(ch)
	}()
	runtime.Gosched()

	select {
	case <-r.ctx.Done():
		return 0, r.ctx.Err()

	case err = <-ch:
		if err != nil {
			return
		}
	}

	if r.br.Buffered() > 0 {
		n, _ = r.br.Read(b)
	}
	return
}
