package main

import (
	"log"
	"net/http"
	"time"

	"private-ethereum-voting/backend/config"
	"private-ethereum-voting/backend/eth"
	"private-ethereum-voting/backend/handlers"
)

func main() {
	// Backend стартует после Geth в Docker Compose, но сама RPC-служба Geth
	// может быть готова не мгновенно. Поэтому подключение выполняется с retry.
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	client, err := eth.DialWithRetry(cfg.RPCURL, 60, time.Second)
	if err != nil {
		log.Fatalf("ethereum client: %v", err)
	}
	defer client.Close()

	app := handlers.NewApp(cfg, client)
	mux := http.NewServeMux()

	// Все HTTP endpoint-ы frontend-а регистрируются в handlers.App.
	app.RegisterRoutes(mux)

	server := &http.Server{
		Addr:         ":8080",
		Handler:      cors(mux),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	log.Println("backend listening on :8080")
	log.Fatal(server.ListenAndServe())
}

func cors(next http.Handler) http.Handler {
	// Frontend раздается Nginx на порту 3000, backend работает на 8080.
	// CORS нужен, чтобы browser разрешил frontend-у вызывать REST API.
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
