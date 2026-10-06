// Command server runs the express-workflow example on port 4242.
package main

import (
	"log"
	"net"
	"os"
	"strconv"

	expressworkflow "github.com/nguyenvanduocit/go-xstate/examples/server/workflow"
)

func main() {
	ln, err := net.Listen("tcp", ":"+strconv.Itoa(expressworkflow.Port))
	if err != nil {
		log.Fatal(err)
	}
	log.Fatal(expressworkflow.Run(os.Stdout, ln, expressworkflow.NewServer(nil).Handler()))
}
