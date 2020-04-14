package main

import (
	"bufio"
	"context"
	"io"
	"net/http"
	"runtime"
	"strconv"
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
		dst.Del(k)

		for _, v := range vr {
			dst.Add(k, v)
		}
	}
}

func (s *streamingRespiratorServer) copy(client io.ReadWriter, remote io.ReadWriter) {
	ctx, ctxCancel := context.WithCancel(context.Background())

	clientWithContext := readerWithContext{
		br:  bufio.NewReader(client),
		ctx: ctx,
	}
	remoteWithContext := readerWithContext{
		br:  bufio.NewReader(remote),
		ctx: ctx,
	}

	done := make(chan struct{}, 2)
	go s.copyOneway(client, &remoteWithContext, done, ctxCancel)
	go s.copyOneway(remote, &clientWithContext, done, ctxCancel)
	<-done
	<-done
}
func (s *streamingRespiratorServer) copyOneway(dst io.Writer, src io.Reader, ch chan struct{}, cancel context.CancelFunc) {
	io.Copy(dst, src)
	cancel()
	ch <- struct{}{}
}

type readerWithContext struct {
	br  *bufio.Reader
	ctx context.Context
}

func (r *readerWithContext) Read(b []byte) (n int, err error) {
	ch := make(chan error)
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
