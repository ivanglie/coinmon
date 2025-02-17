package main

import (
	"os"

	"github.com/ivanglie/coinmon/internal/server"
	"github.com/ivanglie/coinmon/pkg/log"
)

func main() {
	log.SetDefaultLogConfig()

	server := server.New(":8080")

	log.Info("Starting server on :8080")
	if err := server.Start(); err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
}
