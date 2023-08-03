package main

import (
	"flag"
	"log"

	"github.com/zknill/sequenceprotocol/pkg/server"
)

var (
	port = flag.Int("port", 8080, "port to listen server on")
)

func main() {
	flag.Parse()

	if err := server.Listen(*port); err != nil {
		log.Fatal(err)
	}
}
