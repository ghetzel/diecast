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

type mockFile struct {
	fileutil.FileInfo
	file   http.File
	data   []byte
	buf    *bytes.Reader
	header http.Header
}

// Load data from a variety of sources and expose it with an http.File interface.
func newMockFile(name string, src interface{}) (*mockFile, error) {
	var file = new(mockFile)

	file.SetIsDir(false)
	file.SetName(name)

	return file, file.SetSource(src)
}

// setup seekable internal buffer and recalculate size
func (self *mockFile) prep() {
	if self.buf == nil {
		self.buf = bytes.NewReader(self.data)
	}

	self.SetSize(int64(self.buf.Len()))
}

func (self *mockFile) SetHeader(key string, value interface{}) {
	if self.header == nil {
		self.header = make(http.Header)
	}

	self.header.Set(key, typeutil.String(value))
}

func (self *mockFile) Header() http.Header {
	return self.header
}

func (self *mockFile) SetSource(src interface{}) error {
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

func (self *mockFile) SetData(b []byte) {
	self.data = b
	self.prep()
}

func (self *mockFile) Read(b []byte) (int, error) {
	self.prep()

	return self.buf.Read(b)
}

func (self *mockFile) Seek(offset int64, whence int) (int64, error) {
	self.prep()

	return self.buf.Seek(offset, whence)
}

func (self *mockFile) Close() error {
	self.data = nil
	self.buf = nil
	return nil
}

func (self *mockFile) Readdir(count int) ([]os.FileInfo, error) {
	return nil, os.ErrInvalid
}

func (self *mockFile) Stat() (os.FileInfo, error) {
	return &self.FileInfo, nil
}

func (self *mockFile) String() string {
	return string(self.data)
}
