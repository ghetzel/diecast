package diecast

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/beevik/etree"
	"github.com/ghetzel/go-stockutil/maputil"
	"github.com/ghetzel/go-stockutil/typeutil"
)

var ooxmlTemplatedElements = etree.MustCompilePath(`//`)
var etreeMaxAncestors = 3

type OOXMLRenderer struct {
	server *Server
}

func (self *OOXMLRenderer) ShouldPrerender() bool {
	return false
}

func (self *OOXMLRenderer) SetServer(server *Server) {
	self.server = server
}

func (self *OOXMLRenderer) SetPrewriteFunc(fn PrewriteFunc) {}

func (self *OOXMLRenderer) Render(w http.ResponseWriter, req *http.Request, options RenderOptions) error {
	if options.Input != nil {
		defer options.Input.Close()

		var inputExt = strings.ToLower(filepath.Ext(options.RequestedPath))
		var inputType string
		var ooxml bool

		switch inputExt {
		case `.docx`:
			inputType = `application/vnd.openxmlformats-officedocument.wordprocessingml.document`
			ooxml = true
		case `.pptx`:
			inputType = `application/vnd.openxmlformats-officedocument.presentationml.presentation`
			ooxml = true
		case `.xlsx`:
			inputType = `application/vnd.openxmlformats-officedocument.spreadsheetml.sheet`
			ooxml = true
		}

		if ooxml {
			var input io.ReaderAt
			var ierr error
			var zlen int64

			if ra, ok := options.Input.(io.ReaderAt); ok {
				input = ra
			} else if inputData, err := ioutil.ReadAll(options.Input); err == nil {
				var r = bytes.NewReader(inputData)

				input = r
				zlen = int64(r.Len())
			} else {
				ierr = err
			}

			if ierr == nil {
				var output = zip.NewWriter(w)

				// its all gas and no brakes from here:
				// files are read from the zip, rendered, and written to the response in one go
				if inzip, err := zip.NewReader(input, zlen); err == nil {
					w.Header().Set(`Content-Type`, inputType)
					w.Header().Set(`Content-Disposition`, fmt.Sprintf(
						"attachment; filename=%q",
						filepath.Base(options.RequestedPath),
					))

					// for each file in the incoming zip...
					for _, inpart := range inzip.File {
						// generate a corresponding entry in the outgoing zip
						if dest, err := output.CreateHeader(&zip.FileHeader{
							Name:     inpart.Name,
							Comment:  inpart.Comment,
							Method:   inpart.Method,
							Modified: inpart.Modified,
							Extra:    inpart.Extra,
						}); err == nil {
							// open the input file for reading
							if src, err := inpart.Open(); err == nil {
								defer src.Close()

								var terr error

								// process the files inside of the bundle in various ways
								switch ext := strings.ToLower(filepath.Ext(inpart.Name)); ext {
								case `.xml`: // XML documents are parsed as such, with templates in element text being detected and processed
									var doc = etree.NewDocument()

									var _, err = doc.ReadFrom(src)
									src.Close()

									if err == nil {
										var multiFirst *etree.Element
										var tmpl string
										var overrideMap = make(map[string]interface{})

										if overrides := maputil.M(options.Data).Get(`page.renderers.ooxml`).MapNative(); len(overrides) > 0 {
											if oc, err := maputil.CoalesceMap(overrides, `/`); err == nil {
												for xpath, value := range oc {
													if !typeutil.IsEmpty(value) {
														xpath = `/` + strings.TrimPrefix(xpath, `/`)
														overrideMap[xpath] = value
													}
												}
											}
										}

										// find templated elements and do some preprocessing
										for _, el := range doc.Root().FindElementsPath(ooxmlTemplatedElements) {
											var txt = el.Text()
											var flushTmpl bool

											if ot, ok := overrideMap[el.GetPath()]; ok {
												el.SetText(typeutil.String(ot))
											}

											// if el.Text() != `` {
											// 	log.Debugf("OOXML: % -64s", el.GetPath())
											// 	log.Debugf("OOXML:   %s %q", el.FullTag(), el.Text())
											// }

											if multiFirst == nil { // initializers
												// "index of the first occurrence of '{{' is >= 0 AND occurs before the last occurrence of '}}'"
												// aka: single inline template
												if open := strings.Index(txt, `{{`); open >= 0 && open < strings.LastIndex(txt, `}}`) {
													multiFirst = el
													tmpl = txt
													flushTmpl = true
												} else if strings.Contains(txt, `{{`) { // template open occurs in this element, but closes in another later element
													multiFirst = el
													tmpl = txt
												}
											} else {
												// NOTE: this will work whether "{{" appears in the text or not
												if strings.LastIndex(txt, `{{`) < strings.LastIndex(txt, `}}`) { // terminator
													tmpl += txt
													flushTmpl = true
												} else { // intermediate values
													tmpl += txt
												}

												// clear out what's here, as the final rendered value will be placed inside
												// the element that housed the opening "{{"
												el.SetText(``)
											}

											if flushTmpl && multiFirst != nil {
												multiFirst.SetText(tmpl)
												multiFirst = nil
												tmpl = ``
											}
										}

										var intermediate bytes.Buffer

										if _, err := doc.WriteTo(&intermediate); err == nil {
											if eval, err := EvalInline(
												intermediate.String(),
												maputil.DeepCopy(options.Data),
												options.FunctionSet,
											); err == nil {
												_, terr = dest.Write([]byte(eval))
											} else {
												return fmt.Errorf("bad template %q: %v", inpart.Name, err)
											}
										} else {
											return fmt.Errorf("bad intermediate: %v", err)
										}
									} else {
										return fmt.Errorf("bad xml: %v", err)
									}
								default:
									_, terr = io.Copy(dest, src)
								}

								if terr == nil {
									continue
								} else {
									return terr
								}
							} else {
								return fmt.Errorf("bad part %q: %v", inpart.Name, err)
							}
						} else {
							return fmt.Errorf("create header: %v", err)
						}
					}

					return output.Close()
				} else {
					return fmt.Errorf("bad archive: %v", err)
				}
			} else {
				return ierr
			}
		} else {
			return fmt.Errorf("unsupported input type %q", inputType)
		}
	} else {
		return fmt.Errorf("empty input")
	}
}
func isDescendantOrSibling(parent *etree.Element, candidate *etree.Element) bool {
	if candidate == parent {
		return true
	} else if elp := candidate.Parent(); elp != nil {
		for _, c := range elp.ChildElements() {
			if c == parent {
				return true
			}
		}

		return isDescendantOrSibling(parent, elp)
	}

	return false
}
