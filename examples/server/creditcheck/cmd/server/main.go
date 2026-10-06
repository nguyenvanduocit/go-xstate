// Command server runs the mongodb-credit-check-api example on port 4242.
// Set MONGODB_URI to the connection string (JS: the placeholder in actorService.ts),
// for example mongodb://localhost:27017/creditCheck.
package main

import (
	"context"
	"log"
	"net"
	"os"
	"strconv"

	api "github.com/nguyenvanduocit/go-xstate/examples/server/creditcheck"
)

func main() {
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		log.Fatal("MONGODB_URI is not set")
	}
	store, err := api.ConnectMongo(context.Background(), uri)
	if err != nil {
		log.Fatalf("Error connecting to the db... %v", err)
	}
	defer store.Close(context.Background())

	services := api.NewServices(store, api.Env{}, func(line string) { log.Println(line) })
	srv := api.NewServer(api.Machine(services), store, nil)
	defer srv.Close()

	ln, err := net.Listen("tcp", ":"+strconv.Itoa(api.Port))
	if err != nil {
		log.Fatal(err)
	}
	log.Fatal(api.Run(os.Stdout, ln, srv.Handler()))
}
