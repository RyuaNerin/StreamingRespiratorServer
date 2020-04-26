package main

import (
	"net/http"
	"time"

	"github.com/getsentry/sentry-go"
)

func init() {
	err := sentry.Init(
		sentry.ClientOptions{
			Dsn: "https://a02806965cae487ebb3e3ebe50ba3312@sentry.ryuar.in/16",
			HTTPTransport: &http.Transport{
				ExpectContinueTimeout: 30 * time.Second,
				TLSHandshakeTimeout:   30 * time.Second,
				IdleConnTimeout:       30 * time.Second,
				ResponseHeaderTimeout: 30 * time.Second,
				MaxIdleConnsPerHost:   16,
			},
		},
	)
	if err != nil {
		panic(err)
	}
}
