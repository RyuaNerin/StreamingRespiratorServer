package main

import (
	"io"
	"net/http"
	"net/http/httputil"
)

func newHttpMux(withAuth bool) http.Handler {
	mux := http.NewServeMux()

	var auth func(http.Handler) http.Handler
	if withAuth {
		auth = func(handler http.Handler) http.Handler {
			return checkAuth(handler)
		}
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
				w.WriteHeader(http.StatusUnauthorized)
				w.Header().Set("Authenticate", "Basic realm=\"Access to Streamning-Respirator\"")
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

	var wr io.Writer = w
	if chunked {
		wr = httputil.NewChunkedWriter(w)

		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}

		if c, ok := w.(http.CloseNotifier); ok {
			go func() {
				<-c.CloseNotify()
				resp.Body.Close()
			}()
		}
	}
	io.Copy(wr, resp.Body)
}
