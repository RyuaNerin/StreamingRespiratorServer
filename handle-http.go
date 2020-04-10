package main

import "net/http"

func initMux(mux *http.ServeMux) {
	mux.Handle("/api.twitter.com/", CheckAuth(http.StripPrefix("/api.twitter.com/", http.HandlerFunc(handleHttpStreaming))))
	mux.Handle("/userstream.twitter.com/", CheckAuth(http.StripPrefix("/userstream.twitter.com/", http.HandlerFunc(handleHttpStreaming))))

	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	})
}

func handleHttpApi(w http.ResponseWriter, r *http.Request) {

}

func handleHttpStreaming(w http.ResponseWriter, r *http.Request) {

}
