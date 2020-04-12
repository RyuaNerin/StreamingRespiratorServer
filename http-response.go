package main

import (
	"bytes"
	"io/ioutil"
	"net/http"
)

func newResponse(req *http.Request, statusCode int) *http.Response {
	return &http.Response{
		Status:           http.StatusText(statusCode),
		StatusCode:       statusCode,
		Proto:            req.Proto,
		ProtoMajor:       req.ProtoMajor,
		ProtoMinor:       req.ProtoMinor,
		Request:          req,
		TransferEncoding: req.TransferEncoding,
		ContentLength:    0,
	}
}

func newResponseWithText(req *http.Request, statusCode int, responseBody []byte) *http.Response {
	return &http.Response{
		Status:           http.StatusText(statusCode),
		StatusCode:       statusCode,
		Proto:            req.Proto,
		ProtoMajor:       req.ProtoMajor,
		ProtoMinor:       req.ProtoMinor,
		Request:          req,
		TransferEncoding: req.TransferEncoding,
		Body:             ioutil.NopCloser(bytes.NewReader(responseBody)),
		ContentLength:    int64(len(responseBody)),
		Header: http.Header{
			"Content-Type": []string{"text/plain; charset=utf-8"},
		},
	}
}

func newResponseFromResponse(req *http.Request, resp *http.Response, body *bytes.Buffer) *http.Response {
	resp.Proto = req.Proto
	resp.ProtoMajor = req.ProtoMajor
	resp.ProtoMinor = req.ProtoMinor
	resp.Request = req
	resp.TransferEncoding = req.TransferEncoding
	resp.Body = newResponseBuffer(body)
	resp.ContentLength = int64(body.Len())
	resp.Uncompressed = true

	if resp.Header != nil {
		resp.Header.Del("Content-Encoding")
	}

	return resp
}

type ResponseBuffer struct {
	b  *bytes.Buffer
	br *bytes.Reader
}

func newResponseBuffer(b *bytes.Buffer) *ResponseBuffer {
	return &ResponseBuffer{
		b:  b,
		br: bytes.NewReader(b.Bytes()),
	}
}

func (rb *ResponseBuffer) Read(p []byte) (int, error) {
	read, err := rb.br.Read(p)
	return read, err
}

func (rb *ResponseBuffer) Close() error {
	BytesPool.Put(rb.b)
	return nil
}
