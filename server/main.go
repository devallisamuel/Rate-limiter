package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/devallisamuel/Rate-limiter/limiter"
)

func main() {
	// Allow 5 requests per 10 seconds, per client IP.
	store := limiter.NewStore(5, 10*time.Second)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "hello, you got through")
	})

	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", store.Middleware(mux)))
}
