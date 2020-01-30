package diecast

import (
	"bytes"
	"encoding/csv"
	"encoding/xml"
	"fmt"
	"io"
	"net"
	"net/http"
	"sort"
	"strings"

	"github.com/ghetzel/go-stockutil/log"
	"github.com/ghetzel/go-stockutil/maputil"
	"github.com/ghetzel/go-stockutil/typeutil"
	"github.com/jbenet/go-base58"
	"github.com/lucas-clemente/quic-go/http3"
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
	out := make(map[string]interface{})

	attrs := make(map[string]interface{})
	children := make(map[string]interface{})

	for _, attr := range node.Attrs {
		attrs[xmlNtoS(attr.Name)] = typeutil.Auto(attr.Value)
	}

	for _, child := range node.Children {
		key := xmlNtoS(child.XMLName)
		value := xmlNodeToMap(&child)

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
	recs := make([][]interface{}, 0)

	out := map[string]interface{}{
		`headers`: make([]string, 0),
		`records`: recs,
	}

	reader := csv.NewReader(bytes.NewBuffer(data))
	reader.Comma = delim

	if records, err := reader.ReadAll(); err == nil {
		for i, row := range records {
			if i == 0 {
				out[`headers`] = row
			} else {
				outrec := make([]interface{}, len(row))

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

type h3serveable struct {
	*http3.Server
}

func (self *h3serveable) Serve(l net.Listener) error {
	if pc, ok := l.(net.PacketConn); ok {
		return self.Server.Serve(pc)
	} else {
		return fmt.Errorf("a packet-oriented transport is required")
	}
}

func (self *h3serveable) ServeTLS(l net.Listener, _ string, _ string) error {
	return self.Serve(l)
}

func fancyMapJoin(in interface{}) string {
	m := maputil.M(in)

	// extract formatting directives from map keys, which uses in-band signalling that i don't *love*,
	// but it seems simpler-for-the-user than the alternatives right now.
	kvjoin := m.String(`_kvjoin`, `=`)
	vvjoin := m.String(`_join`, `&`)
	kformat := m.String(`_kformat`, "%v")
	vformat := m.String(`_vformat`, "%v")

	data := m.MapNative()

	delete(data, `_kvjoin`)
	delete(data, `_join`)
	delete(data, `_kformat`)
	delete(data, `_vformat`)

	// rekey according to the formatting directives
	for kk, vv := range data {
		delete(data, kk)
		data[fmt.Sprintf(kformat, kk)] = fmt.Sprintf(vformat, vv)
	}

	var pairs []string

	for item := range maputil.M(data).Iter() {
		pairs = append(pairs, fmt.Sprintf("%s%s%v", item.K, kvjoin, item.V))
	}

	sort.Strings(pairs)

	return strings.Join(pairs, vvjoin)
}
