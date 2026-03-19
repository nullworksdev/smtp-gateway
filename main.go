package main

import (
	"log"
	"net/http"
	"os"

	"smtp-gateway/handlers"
	"smtp-gateway/middleware"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/send", handlers.SendEmail)
	mux.HandleFunc("GET /api/health", handlers.Health)

	handler := middleware.Logging(middleware.JSON(mux))

	log.Printf("Email relay listening on :%s\n", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
