package server

import (
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"syscall"
	"time"

	"github.com/zknill/sequenceprotocol/pkg/sequence"
)

func Listen(port int) error {
	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		return fmt.Errorf("start listener: %w", err)
	}

	defer lis.Close()

	clients := make(map[string]*client)

	for {
		conn, err := lis.Accept()
		if err != nil &&
			!errors.Is(err, io.EOF) &&
			!errors.Is(err, io.ErrUnexpectedEOF) {

			return fmt.Errorf("accept: %w", err)
		}

		connMsg := sequence.Connect{}

		if err := connMsg.Decode(conn); err != nil {
			log.Println("failed to decode connection message: %w\n", err)
			conn.Close()

			continue
		}

		// resume clients that are already known to us
		if c, ok := clients[connMsg.ClientID]; ok {
			log.Printf("resuming client: %q\n", connMsg.ClientID)
			c.conn = conn

			go c.serve()

			continue
		}

		log.Printf("starting client: %q\n", connMsg.ClientID)

		// start and register a new client
		c := &client{
			conn:     conn,
			clientID: connMsg.ClientID,
			series:   nRandom(connMsg.N),
			acks:     make([]bool, connMsg.N+1),
		}

		clients[connMsg.ClientID] = c

		go c.serve()
	}
}

func nRandom(n uint32) []uint32 {
	out := make([]uint32, n)

	for i := uint32(0); i < n; i++ {
		out[i] = rand.Uint32()
	}

	return out
}

type client struct {
	conn     io.ReadWriteCloser
	clientID string

	series []uint32
	acks   []bool
}

func (c *client) serve() {
	defer c.conn.Close()

	acks := make(chan uint32)
	go listenAcks(c.conn, acks)

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	seq := 0

	for {
		select {
		case ack := <-acks:
			c.acks[ack] = true

		case <-ticker.C:
			log.Printf("seq [%d]\n", seq)
			if seq > len(c.series) {
				// completed an iteration
				// find unacked messages for resend
				seq = c.unacked()

				if seq == -1 {
					// all acked
					log.Printf("client complete: %q\n", c.clientID)

					return
				}
			}

			if c.acks[seq] {
				// already acked
				seq++

				continue
			}

			if seq == len(c.series) {
				// send checksum

				chk := sequence.Checksum{
					Sequence: uint32(seq),
					Checksum: sequence.CalculateChecksum(c.series),
				}

				if _, err := c.conn.Write(chk.Encode()); err != nil {
					if errors.Is(err, syscall.EPIPE) {
						log.Printf("client complete: %q\n", c.clientID)
						// client has closed the connection, it's done with us
						return
					}

					log.Fatalf("%+T", err)
				}

				seq++

				continue
			}

			n := sequence.Number{
				Sequence: uint32(seq),
				Number:   c.series[seq],
			}

			if _, err := c.conn.Write(n.Encode()); err != nil {
				if errors.Is(err, syscall.EPIPE) {
					log.Printf("client lost: %q\n", c.clientID)

					return
				}

				log.Fatal(err)
			}

			seq++
		}
	}
}

func listenAcks(r io.Reader, acks chan uint32) {
	var (
		err error
		ack sequence.Acknowledge
	)
	for {
		err = ack.Decode(r)
		if err != nil {
			// ignore
			// TODO: but for re-connection requests?

			continue
		}

		acks <- ack.Sequence
	}
}

func (c *client) unacked() int {
	for i := range c.acks {
		if !c.acks[i] {
			return i
		}
	}

	return -1
}
