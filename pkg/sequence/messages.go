package sequence

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
)

type Connect struct {
	N        uint32
	ClientID string
}

func (c *Connect) Decode(r io.Reader) error {
	b := make([]byte, 3+4+4)

	if _, err := io.ReadFull(r, b); err != nil {
		return fmt.Errorf("read header: %w", err)
	}

	if !bytes.Equal(b[0:3], []byte("CON")) {
		return errors.New("not a CON message")
	}

	n := binary.BigEndian.Uint32(b[3:7])
	l := binary.BigEndian.Uint32(b[7:11])

	clientID := make([]byte, l)

	if _, err := io.ReadFull(r, clientID); err != nil {
		return fmt.Errorf("read client id: %w", err)
	}

	c.N = n
	c.ClientID = string(clientID)

	return nil
}

func (c *Connect) Encode() []byte {
	t := []byte("CON")

	n := make([]byte, 4)
	binary.BigEndian.PutUint32(n, c.N)

	l := make([]byte, 4)
	binary.BigEndian.PutUint32(l, uint32(len(c.ClientID)))

	out := append(t, n...)
	out = append(out, l...)
	out = append(out, []byte(c.ClientID)...)

	return out
}

type Number struct {
	Sequence uint32
	Number   uint32
}

func (n *Number) Decode(r io.Reader) error {
	b := make([]byte, 3+4+4)

	if _, err := io.ReadFull(r, b); err != nil {
		return fmt.Errorf("read header: %w", err)
	}

	if !bytes.Equal(b[0:3], []byte("NUM")) {
		return errors.New("not a NUM message")
	}

	s := binary.BigEndian.Uint32(b[3:7])
	p := binary.BigEndian.Uint32(b[7:11])

	n.Sequence = s
	n.Number = p

	log.Printf("[%d] number: %d", n.Sequence, n.Number)

	return nil
}

func (n *Number) Encode() []byte {
	t := []byte("NUM")

	s := make([]byte, 4)
	binary.BigEndian.PutUint32(s, n.Sequence)

	p := make([]byte, 4)
	binary.BigEndian.PutUint32(p, n.Number)

	out := append(t, s...)
	out = append(out, p...)

	return out
}

type Acknowledge struct {
	Sequence uint32
}

func (a *Acknowledge) Decode(r io.Reader) error {
	b := make([]byte, 3+4)

	if _, err := io.ReadFull(r, b); err != nil {
		return fmt.Errorf("read header: %w", err)
	}

	if !bytes.Equal(b[0:3], []byte("ACK")) {
		return errors.New("not a ACK message")
	}

	a.Sequence = binary.BigEndian.Uint32(b[3:7])

	return nil
}

func (a *Acknowledge) Encode() []byte {
	t := []byte("ACK")

	s := make([]byte, 4)
	binary.BigEndian.PutUint32(s, a.Sequence)

	out := append(t, s...)

	return out
}

type Checksum struct {
	Sequence uint32
	Checksum []byte
}

func (c *Checksum) Decode(r io.Reader) error {
	b := make([]byte, 3+4+4)

	if _, err := io.ReadFull(r, b); err != nil {
		return fmt.Errorf("read header: %w", err)
	}

	if !bytes.Equal(b[0:3], []byte("CHK")) {
		return errors.New("not a CHK message")
	}

	s := binary.BigEndian.Uint32(b[3:7])
	l := binary.BigEndian.Uint32(b[7:11])

	chk := make([]byte, l)

	if _, err := io.ReadFull(r, chk); err != nil {
		return fmt.Errorf("read checksum: %w", err)
	}

	c.Sequence = s
	c.Checksum = chk

	log.Printf("[%d] checksum: %x", c.Sequence, c.Checksum)

	return nil
}

func (c *Checksum) Encode() []byte {
	t := []byte("CHK")

	s := make([]byte, 4)
	binary.BigEndian.PutUint32(s, c.Sequence)

	l := make([]byte, 4)
	binary.BigEndian.PutUint32(l, uint32(len(c.Checksum)))

	out := append(t, s...)
	out = append(out, l...)
	out = append(out, c.Checksum...)

	return out
}
