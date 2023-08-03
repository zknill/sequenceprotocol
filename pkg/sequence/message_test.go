package sequence_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zknill/sequenceprotocol/pkg/sequence"
)

func TestConnect(t *testing.T) {
	t.Parallel()

	c := sequence.Connect{
		N:        291,
		ClientID: "a9bfd478-fbe4-4fad-b6ac-6c69c0729192",
	}

	got := sequence.Connect{}
	b := bytes.NewBuffer(c.Encode())

	assert.NoError(t, got.Decode(b))
	assert.Equal(t, c, got)
}

func TestNumber(t *testing.T) {
	t.Parallel()

	n := sequence.Number{
		Sequence: 12,
		Number:   29139,
	}

	got := sequence.Number{}
	b := bytes.NewBuffer(n.Encode())

	assert.NoError(t, got.Decode(b))
	assert.Equal(t, n, got)
}

func TestAcknowledge(t *testing.T) {
	t.Parallel()

	a := sequence.Acknowledge{
		Sequence: 12,
	}

	got := sequence.Acknowledge{}
	b := bytes.NewBuffer(a.Encode())

	assert.NoError(t, got.Decode(b))
	assert.Equal(t, a, got)
}

func TestChecksum(t *testing.T) {
	t.Parallel()

	c := sequence.Checksum{
		Sequence: 19,
		Checksum: []byte("ce236f40a35e48f51e921ad5d28cf320265f33b3"),
	}

	got := sequence.Checksum{}
	b := bytes.NewBuffer(c.Encode())

	assert.NoError(t, got.Decode(b))
	assert.Equal(t, c, got)
}
