package main

import (
	"net/http"
	"os"

	"github.com/ivanglie/coinmon/internal/server"
	"github.com/ivanglie/coinmon/pkg/log"
)

func main() {
	server := server.New()
	http.HandleFunc("/api/v1/spot/", server.HandleSpot)

	log.Info("Starting server on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
}
