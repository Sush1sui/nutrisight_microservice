package server

import "net/http"

func NewRouter() http.Handler {
	mux := http.NewServeMux()
	
	mux.HandleFunc("/", IndexHandler)
	mux.HandleFunc("/barcode", BarcodeHandler)
	
	return mux
}