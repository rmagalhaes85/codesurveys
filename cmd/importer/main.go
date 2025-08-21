package main

import (
	"log"
	_ "os"

	"github.com/rmagalhaes85/codesurveys/internal/importer"
)

func main() {
	if err := importer.ImportExperiment("/tmp/experiment1/"); err != nil {
		log.Fatalf("Error: %v", err)
	}

}
