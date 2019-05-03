// Code generated by "esc -o static.go -pkg diecast -modtime 1500000000 -prefix ui ui"; DO NOT EDIT.

package diecast

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"sync"
	"time"
)

type _escLocalFS struct{}

var _escLocal _escLocalFS

type _escStaticFS struct{}

var _escStatic _escStaticFS

type _escDirectory struct {
	fs   http.FileSystem
	name string
}

type _escFile struct {
	compressed string
	size       int64
	modtime    int64
	local      string
	isDir      bool

	once sync.Once
	data []byte
	name string
}

func (_escLocalFS) Open(name string) (http.File, error) {
	f, present := _escData[path.Clean(name)]
	if !present {
		return nil, os.ErrNotExist
	}
	return os.Open(f.local)
}

func (_escStaticFS) prepare(name string) (*_escFile, error) {
	f, present := _escData[path.Clean(name)]
	if !present {
		return nil, os.ErrNotExist
	}
	var err error
	f.once.Do(func() {
		f.name = path.Base(name)
		if f.size == 0 {
			return
		}
		var gr *gzip.Reader
		b64 := base64.NewDecoder(base64.StdEncoding, bytes.NewBufferString(f.compressed))
		gr, err = gzip.NewReader(b64)
		if err != nil {
			return
		}
		f.data, err = ioutil.ReadAll(gr)
	})
	if err != nil {
		return nil, err
	}
	return f, nil
}

func (fs _escStaticFS) Open(name string) (http.File, error) {
	f, err := fs.prepare(name)
	if err != nil {
		return nil, err
	}
	return f.File()
}

func (dir _escDirectory) Open(name string) (http.File, error) {
	return dir.fs.Open(dir.name + name)
}

func (f *_escFile) File() (http.File, error) {
	type httpFile struct {
		*bytes.Reader
		*_escFile
	}
	return &httpFile{
		Reader:   bytes.NewReader(f.data),
		_escFile: f,
	}, nil
}

func (f *_escFile) Close() error {
	return nil
}

func (f *_escFile) Readdir(count int) ([]os.FileInfo, error) {
	return nil, nil
}

func (f *_escFile) Stat() (os.FileInfo, error) {
	return f, nil
}

func (f *_escFile) Name() string {
	return f.name
}

func (f *_escFile) Size() int64 {
	return f.size
}

func (f *_escFile) Mode() os.FileMode {
	return 0
}

func (f *_escFile) ModTime() time.Time {
	return time.Unix(f.modtime, 0)
}

func (f *_escFile) IsDir() bool {
	return f.isDir
}

func (f *_escFile) Sys() interface{} {
	return f
}

// FS returns a http.Filesystem for the embedded assets. If useLocal is true,
// the filesystem's contents are instead used.
func FS(useLocal bool) http.FileSystem {
	if useLocal {
		return _escLocal
	}
	return _escStatic
}

// Dir returns a http.Filesystem for the embedded assets on a given prefix dir.
// If useLocal is true, the filesystem's contents are instead used.
func Dir(useLocal bool, name string) http.FileSystem {
	if useLocal {
		return _escDirectory{fs: _escLocal, name: name}
	}
	return _escDirectory{fs: _escStatic, name: name}
}

// FSByte returns the named file from the embedded assets. If useLocal is
// true, the filesystem's contents are instead used.
func FSByte(useLocal bool, name string) ([]byte, error) {
	if useLocal {
		f, err := _escLocal.Open(name)
		if err != nil {
			return nil, err
		}
		b, err := ioutil.ReadAll(f)
		_ = f.Close()
		return b, err
	}
	f, err := _escStatic.prepare(name)
	if err != nil {
		return nil, err
	}
	return f.data, nil
}

// FSMustByte is the same as FSByte, but panics if name is not present.
func FSMustByte(useLocal bool, name string) []byte {
	b, err := FSByte(useLocal, name)
	if err != nil {
		panic(err)
	}
	return b
}

// FSString is the string version of FSByte.
func FSString(useLocal bool, name string) (string, error) {
	b, err := FSByte(useLocal, name)
	return string(b), err
}

// FSMustString is the string version of FSMustByte.
func FSMustString(useLocal bool, name string) string {
	return string(FSMustByte(useLocal, name))
}

var _escData = map[string]*_escFile{

	"/autoindex.html": {
		local:   "ui/autoindex.html",
		size:    2202,
		modtime: 1500000000,
		compressed: `
H4sIAAAAAAAC/9xWwW7bOhC85yv2CQ7gAM9inFPgUi7QpgV6SFug7qE3MuLKYiFRCrVFoxD694KSEluK
HaNG00N5kchZz1Izu6T5f1ef3q6+fX4HKeXZ8oR3DwAAniNJiFNpK6Qo+Lp6P7sMeihFqbrXdlpRnSFQ
XWIUEN4Ri6sq2OB+eN7/4aZQNbgB4EdSGJolMtdZvYC8MEVVyhhfDeKak8GU0h08PvdMZnptFpBhQs8y
hJW+xwMkVq/TEYsfpVRKm/WshRdwgfk406M0rNWml41tdONeio1EzsGkhEUE09sKRClAhOIMmq0t83S+
dA50Anh754NFKKBpmHOAWYX966SEpvFLRkHTcJbOl9sUFtj2nORNhvBTK0qjYH5+fjoyjdPQ6M26fbrY
/2D5UebIGaX7I1Z1+XwExJmsqijwDgXLL/r+AOF1oXSiUe2O4my8XR/35MM4DS3ZskYnII3y4k4NPojf
2vMbyqjdQAtKSC0mUfC6jJwDpa2ROXZmBssw5EzuYWX7aH2+Z9GhxscQPdW1l6srvpNdkJVmjTBBQ7b2
5Z7ojNDCVGkLk/IMRPihutJWvJi2paT0e6GNF7fbRugLthXa989g6Ujdu4rpqdrvgaZR2mJMha03Desc
5DpHf26OEm918B+y0DkgnT/muS7Uyk+FTeLLiwuxN9Uhk4/x2BQEL+fzP22yl5eszmHaP+QPKm5qeuT2
hyUEp+E8Cc5AvBG+pc7FoSR/pTw4G52wnLUX0MPl2IGcdX9BfgUAAP//2RsgypoIAAA=
`,
	},

	"/": {
		isDir: true,
		local: "ui",
	},
}
