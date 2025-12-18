package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	var (
		addr      = flag.String("addr", ":8080", "listen address")
		tokenFile = flag.String("token-file", getenv("TOKEN_FILE", "./data/token"), "path to token file")
		initToken = flag.Bool("init-token", false, "generate token file (if missing) and print token")
	)
	flag.Parse()

	store, created, err := NewTokenStore(*tokenFile)
	if err != nil {
		log.Fatalf("token error: %v", err)
	}

	// Init token mode: print token then exit
	if *initToken {
		fmt.Println(store.Get())
		_ = created // kept in case you want different behavior later
		return
	}

	mux := http.NewServeMux()

	// Health (no auth)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":   true,
			"time": time.Now().UTC().Format(time.RFC3339),
		})
	})

	// Auth-protected API
	mux.Handle("/api/system", AuthMiddleware(store, http.HandlerFunc(SystemHandler)))
	mux.Handle("/api/metrics", AuthMiddleware(store, http.HandlerFunc(MetricsHandler)))
	mux.Handle("/api/token/rotate", AuthMiddleware(store, RotateHandler(store)))

	srv := &http.Server{
		Addr:              *addr,
		Handler:           withBasicHardening(mux),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("listening on %s", *addr)
	log.Fatal(srv.ListenAndServe())
}

func getenv(k, def string) string {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	return v
}

func withBasicHardening(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}
