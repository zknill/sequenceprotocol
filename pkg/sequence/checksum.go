package sequence

import (
	"crypto/sha1"
	"encoding/binary"
)

func CalculateChecksum(series []uint32) []byte {
	b := make([]byte, len(series)*4)

	for i := 0; i < len(series); i++ {
		low := i * 4
		high := (i + 1) * 4

		binary.BigEndian.PutUint32(b[low:high], series[i])
	}

	h := sha1.New()
	h.Write(b)

	return h.Sum(nil)
}
