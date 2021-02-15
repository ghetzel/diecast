package diecast

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ghetzel/go-stockutil/fileutil"
	"github.com/ghetzel/go-stockutil/log"
	"github.com/ghetzel/go-stockutil/maputil"
	"github.com/ghetzel/go-stockutil/typeutil"
	"github.com/jbenet/go-base58"
)

func bugWarning() {
	log.Warningf("BUG: no timer associated with request. Please report this at https://github.com/ghetzel/diecast")
}

type xmlNode struct {
	XMLName  xml.Name
	Attrs    []xml.Attr `xml:"-"`
	Content  string     `xml:",chardata"`
	Children []xmlNode  `xml:",any"`
}

func (n *xmlNode) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	n.Attrs = start.Attr
	type node xmlNode

	return d.DecodeElement((*node)(n), &start)
}

func xmlNtoS(name xml.Name) string {
	// if name.Space == `` {
	// 	return
	// } else {
	// 	return fmt.Sprintf("%s:%s", name.Space, name.Local)
	// }
	return name.Local
}

func xmlToMap(in []byte) (map[string]interface{}, error) {
	var docroot xmlNode

	if err := xml.Unmarshal(in, &docroot); err == nil {
		return xmlNodeToMap(&docroot), nil
	} else {
		return nil, err
	}
}

func xmlNodeToMap(node *xmlNode) map[string]interface{} {
	var out = make(map[string]interface{})

	var attrs = make(map[string]interface{})
	var children = make(map[string]interface{})

	for _, attr := range node.Attrs {
		attrs[xmlNtoS(attr.Name)] = typeutil.Auto(attr.Value)
	}

	for _, child := range node.Children {
		var key = xmlNtoS(child.XMLName)
		var value = xmlNodeToMap(&child)

		if existing, ok := children[key]; ok {
			if !typeutil.IsArray(existing) {
				children[key] = append([]interface{}{existing}, value)
			} else if eI, ok := existing.([]interface{}); ok {
				children[key] = append(eI, value)
			}

		} else {
			children[key] = value
		}
	}

	out[`name`] = xmlNtoS(node.XMLName)

	if content := strings.TrimSpace(node.Content); content != `` {
		out[`text`] = content
	}

	out[`attributes`] = attrs

	if len(children) > 0 {
		out[`children`] = children
	}

	return out
}

func xsvToArray(data []byte, delim rune) (map[string]interface{}, error) {
	var recs = make([][]interface{}, 0)

	var out = map[string]interface{}{
		`headers`: make([]string, 0),
		`records`: recs,
	}

	var reader = csv.NewReader(bytes.NewBuffer(data))
	reader.Comma = delim

	if records, err := reader.ReadAll(); err == nil {
		for i, row := range records {
			if i == 0 {
				out[`headers`] = row
			} else {
				var outrec = make([]interface{}, len(row))

				for j, col := range row {
					outrec[j] = typeutil.Auto(col)
				}

				if len(outrec) > 0 {
					recs = append(recs, outrec)
				}
			}
		}

		out[`records`] = recs

		return out, nil
	} else {
		return nil, err
	}
}

type funcHandler struct {
	fn http.HandlerFunc
}

func (self *funcHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	self.fn(w, req)
}

func constantErrHandler(server *Server, err error, code int) http.Handler {
	return &funcHandler{
		fn: func(w http.ResponseWriter, req *http.Request) {
			server.respondError(w, req, err, code)
		},
	}
}

func b58encode(data []byte) string {
	return base58.EncodeAlphabet(data, base58.BTCAlphabet)
}

func b58decode(data string) []byte {
	return base58.DecodeAlphabet(data, base58.BTCAlphabet)
}

type multiReadCloser struct {
	reader  io.Reader
	closers []io.Closer
}

func MultiReadCloser(readers ...io.Reader) *multiReadCloser {
	var closers = make([]io.Closer, 0)

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

type statusInterceptor struct {
	http.ResponseWriter
	code         int
	bytesWritten int64
}

func intercept(upstream http.ResponseWriter) *statusInterceptor {
	return &statusInterceptor{
		ResponseWriter: upstream,
		code:           http.StatusOK,
	}
}

func (self *statusInterceptor) WriteHeader(code int) {
	self.ResponseWriter.WriteHeader(code)
	self.code = code
}

func (self *statusInterceptor) Write(b []byte) (int, error) {
	n, err := self.ResponseWriter.Write(b)
	self.bytesWritten += int64(n)
	return n, err
}

// A do-nothing http.Handler that does nothing
type nopHandler struct{}

func (self *nopHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {}

func fancyMapJoin(in interface{}) string {
	var m = maputil.M(in)

	// extract formatting directives from map keys, which uses in-band signalling that i don't *love*,
	// but it seems simpler-for-the-user than the alternatives right now.
	var kvjoin = m.String(`_kvjoin`, `=`)
	var vvjoin = m.String(`_join`, `&`)
	var kformat = m.String(`_kformat`, "%v")
	var vformat = m.String(`_vformat`, "%v")

	var data = m.MapNative()
	var deletes []string

	delete(data, `_kvjoin`)
	delete(data, `_join`)
	delete(data, `_kformat`)
	delete(data, `_vformat`)

	// rekey according to the formatting directives
	for key, value := range data {
		if newkey := fmt.Sprintf(kformat, key); newkey != key {
			deletes = append(deletes, key)
			key = newkey
		}

		data[key] = fmt.Sprintf(vformat, value)
	}

	for _, dk := range deletes {
		delete(data, dk)
	}

	var pairs []string

	for item := range maputil.M(data).Iter() {
		pairs = append(pairs, fmt.Sprintf("%s%s%s", item.K, kvjoin, item.Value))
	}

	sort.Strings(pairs)

	return strings.Join(pairs, vvjoin)
}

// -------------------------------------------------------------------------------------------------
type zipFS struct {
	archive *zip.Reader
}

func newZipFsFromFile(path string) (*zipFS, error) {
	if file, err := os.Open(path); err == nil {
		var sz, _ = file.Stat()

		if z, err := zip.NewReader(file, sz.Size()); err == nil {
			return newZipFS(z), nil
		} else {
			return nil, err
		}
	} else {
		return nil, err
	}
}

func newZipFS(archive *zip.Reader) *zipFS {
	return &zipFS{
		archive: archive,
	}
}

func (self *zipFS) Open(name string) (http.File, error) {
	name = filepath.Clean(name)

	switch name {
	case ``, `/`, `./`:
		return newZipEntry(self, &zip.File{
			FileHeader: zip.FileHeader{
				Name: `/`,
			},
		}), nil
	default:
		for _, hdr := range self.archive.File {
			if name == filepath.Clean(hdr.Name) {
				return newZipEntry(self, hdr), nil
			}
		}
	}

	return nil, os.ErrNotExist
}

func (self *zipFS) entries(path string) []os.FileInfo {
	path = filepath.Clean(path)
	path = strings.TrimPrefix(path, `/`)

	switch path {
	case `./`, ``:
		path = `/`
	}

	var dirs []os.FileInfo
	var files []os.FileInfo

	for _, hdr := range self.archive.File {
		var dirname = filepath.Dir(filepath.Clean(hdr.Name))

		if (path == `/` && dirname == `.`) || (dirname == path) {
			if info := hdr.FileInfo(); info.IsDir() {
				dirs = append(dirs, info)
			} else {
				files = append(files, info)
			}
		}
	}

	return append(dirs, files...)
}

// -------------------------------------------------------------------------------------------------
type zipEntry struct {
	fs      *zipFS
	zipfile *zip.File
	rs      io.ReadSeeker
	entries []os.FileInfo
	offset  int
}

func newZipEntry(fs *zipFS, file *zip.File) *zipEntry {
	return &zipEntry{
		fs:      fs,
		zipfile: file,
	}
}

func (self *zipEntry) prep() error {
	if self.rs == nil {
		if rc, err := self.zipfile.Open(); err == nil {
			defer rc.Close()

			if data, err := ioutil.ReadAll(rc); err == nil {
				self.rs = bytes.NewReader(data)
			} else {
				return err
			}
		} else {
			return err
		}
	}

	return nil
}

func (self *zipEntry) Read(b []byte) (int, error) {
	if err := self.prep(); err == nil {
		return self.rs.Read(b)
	} else {
		return 0, err
	}
}

func (self *zipEntry) Seek(offset int64, whence int) (int64, error) {
	if err := self.prep(); err == nil {
		return self.rs.Seek(offset, whence)
	} else {
		return 0, err
	}
}

func (self *zipEntry) Close() error {
	self.rs = nil
	self.zipfile = nil
	return nil
}

func (self *zipEntry) Readdir(count int) ([]os.FileInfo, error) {
	if self.zipfile.FileInfo().IsDir() {
		if self.entries == nil {
			self.entries = self.fs.entries(self.zipfile.Name)
		}

		if self.offset <= len(self.entries) {
			if end := (self.offset + count); end < len(self.entries) {
				var err error
				var sub = self.entries[self.offset:end]

				if self.offset >= len(self.entries) {
					err = io.EOF
				}

				self.offset += len(sub)

				return sub, err
			}
		}
	}

	return nil, io.EOF
}

func (self *zipEntry) Stat() (os.FileInfo, error) {
	return self.zipfile.FileInfo(), nil
}

// -------------------------------------------------------------------------------------------------
type httpFile struct {
	*fileutil.FileInfo
	buf    *bytes.Reader
	closed bool
}

func newHttpFile(name string, data []byte) *httpFile {
	var f = &httpFile{
		FileInfo: fileutil.NewFileInfo(nil),
		buf:      bytes.NewReader(data),
	}

	f.SetSize(int64(f.buf.Len()))
	f.SetIsDir(false)
	f.SetName(name)

	return f
}

func (self *httpFile) Close() error {
	self.closed = true
	return nil
}

func (self *httpFile) Read(b []byte) (int, error) {
	if self.closed {
		return 0, fmt.Errorf("attempted read on closed file")
	} else if self.buf == nil {
		return 0, io.EOF
	} else {
		return self.buf.Read(b)
	}
}

func (self *httpFile) Seek(offset int64, whence int) (int64, error) {
	if self.closed {
		return 0, fmt.Errorf("attempted seek on closed file")
	} else if self.buf == nil {
		return 0, io.EOF
	} else {
		return self.buf.Seek(offset, whence)
	}
}

func (self *httpFile) Readdir(count int) ([]os.FileInfo, error) {
	return nil, io.EOF
}

func (self *httpFile) Stat() (os.FileInfo, error) {
	return self.FileInfo, nil
}
