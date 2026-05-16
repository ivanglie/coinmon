// Package main is the entry point of the application.
package main

import (
	"os"

	"github.com/ivanglie/coinmon/internal/server"
	"github.com/ivanglie/coinmon/pkg/log"
)

func main() {
	log.SetDefaultLogConfig()

	s := server.New(":8080")

	log.Info("Starting server on :8080")
	if err := s.Start(); err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
}
