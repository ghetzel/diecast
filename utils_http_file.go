package diecast

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/ghetzel/go-stockutil/fileutil"
	"github.com/ghetzel/go-stockutil/typeutil"
)

type mockHttpFile struct {
	fileutil.FileInfo
	file http.File
	data []byte
	buf  *bytes.Reader
}

// Load data from a variety of sources and expose it with an http.File interface.
func newMockHttpFile(name string, src interface{}) (*mockHttpFile, error) {
	var file = new(mockHttpFile)

	file.SetIsDir(false)
	file.SetName(name)

	return file, file.SetSource(src)
}

// setup seekable internal buffer and recalculate size
func (self *mockHttpFile) prep() {
	if self.buf == nil {
		self.buf = bytes.NewReader(self.data)
	}

	self.SetSize(int64(self.buf.Len()))
}

func (self *mockHttpFile) SetSource(src interface{}) error {
	if src == nil { // nil source
		self.SetData(nil)
		return nil
	} else if f, ok := src.(http.File); ok { // http.File
		defer f.Close()

		if s, err := f.Stat(); err == nil {
			self.SetIsDir(s.IsDir())
			self.SetName(s.Name())
			self.SetMode(s.Mode())
			self.SetModTime(s.ModTime())
			self.SetSys(s.Sys())
		} else {
			return err
		}

		if b, err := ioutil.ReadAll(f); err == nil {
			self.SetData(b)
		} else {
			return err
		}
	} else if r, ok := src.(io.Reader); ok { // io.Reader & io.ReadCloser
		if b, err := ioutil.ReadAll(r); err == nil {
			self.SetData(b)
		} else {
			return err
		}

		if c, ok := src.(io.Closer); ok {
			defer c.Close()
		}
	} else if err, ok := src.(error); ok { // error
		self.SetData([]byte(err.Error()))
	} else {
		self.SetData(typeutil.Bytes(src))
	}

	return nil
}

func (self *mockHttpFile) SetData(b []byte) {
	self.data = b
	self.prep()
}

func (self *mockHttpFile) Read(b []byte) (int, error) {
	self.prep()

	return self.buf.Read(b)
}

func (self *mockHttpFile) Seek(offset int64, whence int) (int64, error) {
	self.prep()

	return self.buf.Seek(offset, whence)
}

func (self *mockHttpFile) Close() error {
	self.data = nil
	self.buf = nil
	return nil
}

func (self *mockHttpFile) Readdir(count int) ([]os.FileInfo, error) {
	return nil, os.ErrInvalid
}

func (self *mockHttpFile) Stat() (os.FileInfo, error) {
	return &self.FileInfo, nil
}

func (self *mockHttpFile) String() string {
	return string(self.data)
}
