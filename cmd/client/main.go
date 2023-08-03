package main

import (
	"flag"
	"log"
	"os"
	"os/signal"

	"github.com/google/uuid"
	"github.com/zknill/sequenceprotocol/pkg/client"
)

var (
	port = flag.Int("port", 8080, "server port to connect to")
	n    = flag.Int("n", 10, "size of the series")
	id   = flag.String("id", uuid.New().String(), "client id to use in the connection, random each time unless set")
)

func main() {
	flag.Parse()

	done := make(chan struct{})

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for _ = range c {
			close(done)
		}
	}()

	if err := client.Sequence(done, *port, *n, *id); err != nil {
		log.Fatal(err)
	}
}
