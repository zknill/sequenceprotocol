package client

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"

	"github.com/zknill/sequenceprotocol/pkg/sequence"
	"github.com/zknill/sequenceprotocol/pkg/store"
)

func Sequence(done chan struct{}, port int, n int, id string) error {
	conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		log.Fatal(err)
	}

	connMsg := sequence.Connect{
		N:        uint32(n),
		ClientID: id,
	}

	// todo, error handling
	if _, err := conn.Write(connMsg.Encode()); err != nil {
		log.Fatal(err)
	}

	series, err := store.New(id, n)
	if err != nil {
		return fmt.Errorf("create store: %w", err)
	}
	defer series.Flush()

	//series := make([]uint32, n)
	//received := make([]bool, n+1)
	chk := sequence.Checksum{}
	acks := make(chan uint32)

	go writeAcks(acks, conn)

	for {
		select {
		case <-done:
			log.Println("exiting")
			return nil
		default:
		}

		if series.AllReceived() {
			conn.Close()

			computed := sequence.CalculateChecksum(series.Series())
			checkSumMatch := bytes.Equal(computed, chk.Checksum)

			if checkSumMatch {
				log.Println("[OK] checksum match")
				log.Printf("%x = %x", chk.Checksum, computed)

				return nil
			}

			log.Println("[FAIL] checksum no match")
			log.Printf("%x != %x", chk.Checksum, computed)
			os.Exit(1)
		}

		match, r, err := peek("NUM", conn)
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				return errors.New("server closed the connection")
			}

			return fmt.Errorf("peek for NUM: %w", err)
		}

		if match {
			n := sequence.Number{}

			if err := n.Decode(r); err != nil {
				log.Fatal(err)
			}

			//series[n.Sequence] = n.Number
			//received[n.Sequence] = true
			series.ReceivedNum(n.Sequence, n.Number)

			go pushAck(n.Sequence, acks)

			continue
		}

		_, r, err = peek("CHK", r)
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				return errors.New("server closed the connection")
			}

			return fmt.Errorf("peek for CHK: %w", err)
		}

		if err := chk.Decode(r); err != nil {
			log.Fatal(err)
		}

		//received[chk.Sequence] = true
		series.ReceivedChecksum(chk.Sequence)

		go pushAck(chk.Sequence, acks)
	}
}

func allReceived(received []bool) bool {
	for _, v := range received {
		if !v {
			return false
		}
	}

	return true
}

func writeAcks(acks chan uint32, w io.Writer) {
	for ack := range acks {
		a := sequence.Acknowledge{
			Sequence: ack,
		}

		w.Write(a.Encode())
	}
}

func pushAck(ack uint32, acks chan uint32) {
	acks <- ack
}

func peek(t string, r io.Reader) (bool, io.Reader, error) {
	b := make([]byte, len(t))

	if _, err := io.ReadFull(r, b); err != nil {
		return false, nil, fmt.Errorf("read: %w", err)
	}

	match := bytes.Equal(b, []byte(t))
	reader := io.MultiReader(bytes.NewBuffer(b), r)

	return match, reader, nil
}
