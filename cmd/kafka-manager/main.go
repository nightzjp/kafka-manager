package main

import (
	"log"
	"net/http"

	"github.com/nightzjp/kafka-manager/internal/app"
)

func main() {
	server := &http.Server{
		Addr:    ":8080",
		Handler: app.New(),
	}
	log.Printf("kafka-manager listening on %s", server.Addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
