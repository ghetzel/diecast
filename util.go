package diecast

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"encoding/xml"
	"fmt"
	"io"
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

func xmlToMap(in []byte) (map[string]any, error) {
	var docroot xmlNode

	if err := xml.Unmarshal(in, &docroot); err == nil {
		return xmlNodeToMap(&docroot), nil
	} else {
		return nil, err
	}
}

func xmlNodeToMap(node *xmlNode) map[string]any {
	var out = make(map[string]any)

	var attrs = make(map[string]any)
	var children = make(map[string]any)

	for _, attr := range node.Attrs {
		attrs[xmlNtoS(attr.Name)] = typeutil.Auto(attr.Value)
	}

	for _, child := range node.Children {
		var key = xmlNtoS(child.XMLName)
		var value = xmlNodeToMap(&child)

		if existing, ok := children[key]; ok {
			if !typeutil.IsArray(existing) {
				children[key] = append([]any{existing}, value)
			} else if eI, ok := existing.([]any); ok {
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

func xsvToArray(data []byte, delim rune) (map[string]any, error) {
	var recs = make([][]any, 0)

	var out = map[string]any{
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
				var outrec = make([]any, len(row))

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

func (multi *multiReadCloser) Read(p []byte) (int, error) {
	return multi.reader.Read(p)
}

func (multi *multiReadCloser) Close() error {
	var mErr error

	for _, closer := range multi.closers {
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

func (intercept *statusInterceptor) WriteHeader(code int) {
	intercept.ResponseWriter.WriteHeader(code)
	intercept.code = code
}

func (intercept *statusInterceptor) Write(b []byte) (int, error) {
	n, err := intercept.ResponseWriter.Write(b)
	intercept.bytesWritten += int64(n)
	return n, err
}

func fancyMapJoin(in any) string {
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

func (zipfs *zipFS) Open(name string) (http.File, error) {
	name = filepath.Clean(name)

	switch name {
	case ``, `/`, `./`:
		return newZipEntry(zipfs, &zip.File{
			FileHeader: zip.FileHeader{
				Name: `/`,
			},
		}), nil
	default:
		for _, hdr := range zipfs.archive.File {
			if name == filepath.Clean(hdr.Name) {
				return newZipEntry(zipfs, hdr), nil
			}
		}
	}

	return nil, os.ErrNotExist
}

func (zipfs *zipFS) entries(path string) []os.FileInfo {
	path = filepath.Clean(path)
	path = strings.TrimPrefix(path, `/`)

	switch path {
	case `./`, ``:
		path = `/`
	}

	var dirs []os.FileInfo
	var files []os.FileInfo

	for _, hdr := range zipfs.archive.File {
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

func (entry *zipEntry) prep() error {
	if entry.rs == nil {
		if rc, err := entry.zipfile.Open(); err == nil {
			defer rc.Close()

			if data, err := io.ReadAll(rc); err == nil {
				entry.rs = bytes.NewReader(data)
			} else {
				return err
			}
		} else {
			return err
		}
	}

	return nil
}

func (entry *zipEntry) Read(b []byte) (int, error) {
	if err := entry.prep(); err == nil {
		return entry.rs.Read(b)
	} else {
		return 0, err
	}
}

func (entry *zipEntry) Seek(offset int64, whence int) (int64, error) {
	if err := entry.prep(); err == nil {
		return entry.rs.Seek(offset, whence)
	} else {
		return 0, err
	}
}

func (entry *zipEntry) Close() error {
	entry.rs = nil
	entry.zipfile = nil
	return nil
}

func (entry *zipEntry) Readdir(count int) ([]os.FileInfo, error) {
	if entry.zipfile.FileInfo().IsDir() {
		if entry.entries == nil {
			entry.entries = entry.fs.entries(entry.zipfile.Name)
		}

		if entry.offset <= len(entry.entries) {
			if end := (entry.offset + count); end < len(entry.entries) {
				var err error
				var sub = entry.entries[entry.offset:end]

				if entry.offset >= len(entry.entries) {
					err = io.EOF
				}

				entry.offset += len(sub)

				return sub, err
			}
		}
	}

	return nil, io.EOF
}

func (entry *zipEntry) Stat() (os.FileInfo, error) {
	return entry.zipfile.FileInfo(), nil
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

func (hfile *httpFile) Close() error {
	hfile.closed = true
	return nil
}

func (hfile *httpFile) Read(b []byte) (int, error) {
	if hfile.closed {
		return 0, fmt.Errorf("attempted read on closed file")
	} else if hfile.buf == nil {
		return 0, io.EOF
	} else {
		return hfile.buf.Read(b)
	}
}

func (hfile *httpFile) Seek(offset int64, whence int) (int64, error) {
	if hfile.closed {
		return 0, fmt.Errorf("attempted seek on closed file")
	} else if hfile.buf == nil {
		return 0, io.EOF
	} else {
		return hfile.buf.Seek(offset, whence)
	}
}

func (hfile *httpFile) Readdir(count int) ([]os.FileInfo, error) {
	return nil, io.EOF
}

func (hfile *httpFile) Stat() (os.FileInfo, error) {
	return hfile.FileInfo, nil
}
