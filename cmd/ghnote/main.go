package main

import (
	"context"
	"log"
	"os"

	"github-note/internal/app"
)

func main() {
	ctx := context.Background()

	if err := app.Run(ctx, os.Args[1:]); err != nil {
		log.Fatalf("ghnote failed: %v", err)
	}
}
