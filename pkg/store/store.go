package store

import (
	"encoding/json"
	"errors"
	"fmt"
	iofs "io/fs"
	"log"
	"os"
	"path"
)

type FS struct {
	path     string
	vals     []uint32
	received []bool
}

func New(id string, n int) (*FS, error) {
	path := path.Join(os.TempDir(), id+".state")

	_, err := os.Stat(path)
	notExists := errors.Is(err, iofs.ErrNotExist)

	fs := &FS{
		path:     path,
		vals:     make([]uint32, n),
		received: make([]bool, n+1),
	}

	if notExists {
		return fs, nil
	}

	return fs, fs.load()
}

func (fs *FS) ReceivedNum(i uint32, v uint32) {
	fs.vals[i] = v
	fs.received[i] = true
}

func (fs *FS) ReceivedChecksum(i uint32) {
	fs.received[i] = true
}

func (fs *FS) Series() []uint32 {
	return fs.vals
}

func (fs *FS) AllReceived() bool {
	for _, v := range fs.received {
		if !v {
			return false
		}
	}

	return true
}

type fileContent struct {
	Vals     []uint32 `json:"vals"`
	Received []bool   `json:"received"`
}

func (fs *FS) Flush() error {
	f, err := os.Create(fs.path)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}

	c := fileContent{
		Vals:     fs.vals,
		Received: fs.received,
	}

	if err := json.NewEncoder(f).Encode(c); err != nil {
		return fmt.Errorf("write state: %w", err)
	}

	return nil
}

func (fs *FS) load() error {
	b, err := os.ReadFile(fs.path)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	c := fileContent{}

	if err := json.Unmarshal(b, &c); err != nil {
		return fmt.Errorf("write state: %w", err)
	}

	fs.vals = c.Vals
	fs.received = c.Received

	log.Printf("resumed data from store, series: %v\n", fs.vals)

	return nil
}
