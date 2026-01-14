package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/rfielding/turducken/pkg/server"
)

func main() {
	port := flag.Int("port", 8080, "HTTP server port")
	specFile := flag.String("spec", "", "Prolog specification file to load")
	flag.Parse()

	// If spec file provided, verify it exists
	if *specFile != "" {
		if _, err := os.Stat(*specFile); os.IsNotExist(err) {
			log.Fatalf("Specification file not found: %s", *specFile)
		}
	}

	srv, err := server.New(*specFile)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	addr := fmt.Sprintf(":%d", *port)
	log.Printf("Starting turducken server on http://localhost%s", addr)
	if *specFile != "" {
		log.Printf("Loaded specification: %s", *specFile)
	}
	
	if err := srv.ListenAndServe(addr); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
