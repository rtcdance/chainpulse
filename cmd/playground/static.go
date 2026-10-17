package main

import (
	_ "embed"
	"net/http"
)

//go:embed index.html
var indexHTML []byte

func handleHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, err := w.Write(indexHTML); err != nil {
		http.Error(w, "failed to write response", http.StatusInternalServerError)
	}
}
