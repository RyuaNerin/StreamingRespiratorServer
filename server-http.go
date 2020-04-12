package main

import (
	"io"
	"net/http"
)

func newHttpMux(withAuth bool) http.Handler {
	mux := http.NewServeMux()

	var auth func(http.Handler) http.Handler
	if withAuth {
		auth = checkAuth
	} else {
		auth = func(handler http.Handler) http.Handler {
			return handler
		}
	}

	mux.Handle("/api.twitter.com/", auth(http.StripPrefix("/api.twitter.com/", http.HandlerFunc(handleHttpApi))))
	mux.Handle("/userstream.twitter.com/1.1/user.json", auth(http.HandlerFunc(handleHttpStreaming(false))))
	mux.Handle("/userstream", auth(http.HandlerFunc(handleHttpStreaming(true))))

	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	})

	if !verbose {
		return mux
	} else {
		return http.HandlerFunc(
			func(w http.ResponseWriter, req *http.Request) {
				logger.Println("START", req.URL.String())
				mux.ServeHTTP(w, req)
				logger.Println("END", req.URL.String())
			},
		)
	}
}

func checkAuth(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		id, pw, ok := splitProxyAuthorizationHeader(req)
		if !ok || id != authId || pw != authPw {
			id, pw, ok = req.BasicAuth()
			if !ok || id != authId || pw != authPw {
				w.Header().Set("WWW-Authenticate", AuthenticateHeaderValue)
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
		}

		handler.ServeHTTP(w, req)
	})
}

func handleHttpStreaming(isHttpServer bool) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		writeResponseToWriter(handleStreaming(req, isHttpServer), w, true)
	}
}

func handleHttpApi(w http.ResponseWriter, req *http.Request) {
	req.URL.Host = "api.twitter.com"
	req.Host = "api.twitter.com"

	writeResponseToWriter(handleApi(req, true), w, false)
}

func writeResponseToWriter(resp *http.Response, w http.ResponseWriter, chunked bool) {
	defer resp.Body.Close()

	h := w.Header()
	for k, vr := range resp.Header {
		for _, v := range vr {
			h.Add(k, v)
		}
	}

	w.WriteHeader(resp.StatusCode)

	if !chunked {
		io.Copy(w, resp.Body)
	} else {
		f, fok := w.(http.Flusher)

		if fok {
			f.Flush()
		}

		if c, ok := w.(http.CloseNotifier); ok {
			go func() {
				<-c.CloseNotify()
				resp.Body.Close()
			}()
		}

		buff := make([]byte, 16*1024)
		for {
			nr, er := resp.Body.Read(buff)
			if nr > 0 {
				nw, ew := w.Write(buff[0:nr])
				if ew != nil {
					break
				}
				if nr != nw {
					break
				}

				if fok {
					f.Flush()
				}
			}
			if er != nil {
				break
			}
		}
	}
}
