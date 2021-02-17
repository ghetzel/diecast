package diecast

import (
	"io"

	"github.com/ghetzel/go-stockutil/log"
)

type multiReadCloser struct {
	mr io.Reader
	mc []io.Closer
}

func MultiReadCloser(readers ...io.Reader) io.ReadCloser {
	var mrc = new(multiReadCloser)

	mrc.mr = io.MultiReader(readers...)

	for _, r := range readers {
		if c, ok := r.(io.Closer); ok {
			mrc.mc = append(mrc.mc, c)
		}
	}

	return mrc
}

func (self *multiReadCloser) Read(b []byte) (int, error) {
	if self.mr == nil {
		return 0, io.EOF
	} else {
		return self.mr.Read(b)
	}
}

func (self *multiReadCloser) Close() error {
	var merr error

	for _, c := range self.mc {
		if c != nil {
			merr = log.AppendError(merr, c.Close())
		}
	}

	return merr
}
