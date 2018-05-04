package util

import "io"

// Represents an io.Reader that takes zero of more readers, reading from
// each one until it is exhausted.  Returns an io.EOF when all readers are read.
type ChainableReader struct {
	readers []io.Reader
	current int
	total   int
}

func NewChainableReader(readers ...io.Reader) *ChainableReader {
	return &ChainableReader{
		readers: readers,
	}
}

func (self *ChainableReader) Read(p []byte) (int, error) {
	if self.current >= 0 && self.current < len(self.readers) {
		if n, err := self.readers[self.current].Read(p); err == nil {
			return n, nil
		} else {
			self.current += 1
			return self.Read(p)
		}
	} else {
		return 0, io.EOF
	}
}

func (self *ChainableReader) Close() error {
	self.current = -1
	self.readers = nil
	return nil
}
