package main

import (
	"log"
	"os"

	"github.com/tnierman/git-grove/cmd"
)

func main() {
	err := cmd.Grove()
	if err != nil {
		log.Printf("ERROR: %v", err)
		os.Exit(1)
	}
}
