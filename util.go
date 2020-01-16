package diecast

import (
	"bytes"
	"encoding/csv"
	"encoding/xml"
	"net/http"
	"strings"

	"github.com/ghetzel/go-stockutil/typeutil"
)

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
