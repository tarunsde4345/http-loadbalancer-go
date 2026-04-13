package main

import (
	"fmt"
	"net/http"
	"os"
	"time"
)

func main() {
	port := os.Getenv("PORT")
	delay := os.Getenv("DELAY") // e.g. "500ms", "1s"

	d, err := time.ParseDuration(delay)
	if err != nil {
		d = 0
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(d) // simulate slow backend
		fmt.Fprintf(w, "backend port=%s delay=%v\n", port, d)
	})

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	fmt.Printf("mock server listening on :%s with delay %v\n", port, d)
	http.ListenAndServe(":"+port, mux)
}