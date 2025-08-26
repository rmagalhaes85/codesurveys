package main

import (
	"context"
	"log"
	"os"

	"github.com/rmagalhaes85/codesurveys/internal/webapp"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("Env var DATABASE_URL must be defined")
	}

	srv, err := webapp.New(context.Background(), dsn)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("listening on http://localhost%s...", srv.Addr)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
