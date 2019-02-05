package diecast

import (
	"io"

	"github.com/ghetzel/go-stockutil/log"
)

type multiReadCloser struct {
	reader  io.Reader
	closers []io.Closer
}

func MultiReadCloser(readers ...io.Reader) *multiReadCloser {
	closers := make([]io.Closer, 0)

	for _, r := range readers {
		if closer, ok := r.(io.Closer); ok {
			closers = append(closers, closer)
		}
	}

	return &multiReadCloser{
		reader:  io.MultiReader(readers...),
		closers: closers,
	}
}

func (self *multiReadCloser) Read(p []byte) (int, error) {
	return self.reader.Read(p)
}

func (self *multiReadCloser) Close() error {
	var mErr error

	for _, closer := range self.closers {
		mErr = log.AppendError(mErr, closer.Close())
	}

	return mErr
}
