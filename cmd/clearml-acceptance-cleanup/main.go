package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/pbronneberg/terraform-provider-clearml/internal/acctestcleanup"
	"github.com/pbronneberg/terraform-provider-clearml/internal/client"
)

func main() {
	ctx := context.Background()
	accessKey := os.Getenv("CLEARML_ACCESS_KEY")
	secretKey := os.Getenv("CLEARML_SECRET_KEY")
	if accessKey == "" || secretKey == "" {
		fmt.Fprintln(os.Stderr, "CLEARML_ACCESS_KEY and CLEARML_SECRET_KEY must be set")
		os.Exit(2)
	}
	apiURL := os.Getenv("CLEARML_API_URL")
	if apiURL == "" {
		apiURL = "https://api.clear.ml"
	}
	c, err := client.NewClearMLClient(ctx, "terraform-provider-clearml/acceptance-cleanup", accessKey, secretKey, apiURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "configure ClearML client: %v\n", err)
		os.Exit(1)
	}
	deleted, err := acctestcleanup.Prune(ctx, c, time.Now(), 24*time.Hour)
	if err != nil {
		fmt.Fprintf(os.Stderr, "clean stale acceptance queues: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Deleted %d stale acceptance queue(s).\n", deleted)
}
