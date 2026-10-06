//go:build mongodb

// Command donut mirrors main.ts: it persists the donut actor in MongoDB after every
// snapshot and restores it on start. Reads events from stdin, one per line.
//
//	MONGODB_URI=mongodb://localhost:27017 go run -tags mongodb ./mongodb-persisted-state/cmd/donut
package main

import (
	"context"
	"fmt"
	"os"

	donut "github.com/nguyenvanduocit/go-xstate/examples/server/persistedstate"
)

func main() {
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		fmt.Fprintln(os.Stderr, "MONGODB_URI is not set")
		os.Exit(2)
	}
	store, err := donut.NewMongoStore(uri)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	donut.Run(context.Background(), os.Stdout, os.Stdin, store)
}
