package server

import (
	"fmt"
	"net/http"
)

func initRoutes() {
	http.HandleFunc("/", index)

	http.HandleFunc("/item", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			getItem(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
			fmt.Fprintf(w, "Method not allowed")
		}

	})
	http.HandleFunc("/sync", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			sincItems(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
			fmt.Fprintf(w, "Method not allowed")
		}
	})
}
