package main

import (
	"fmt"
	"io"
)

type debugStream struct {
	name string
	ioi  interface{}
	rn   int
	wn   int
}

func (drc *debugStream) Read(p []byte) (n int, err error) {
	n, err = drc.ioi.(io.Reader).Read(p)
	fmt.Println(drc.name, "read", n, err)
	drc.rn += n
	return
}
func (drc *debugStream) Write(p []byte) (n int, err error) {
	n, err = drc.ioi.(io.Writer).Write(p)
	fmt.Println(drc.name, "write", n, err)
	drc.wn += n
	return
}
func (drc *debugStream) Close() (err error) {
	err = drc.ioi.(io.Closer).Close()
	fmt.Println(drc.name, "close", err, "read : ", drc.rn, "| write : ", drc.wn)
	return
}
