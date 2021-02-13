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
)

var allDescendants = etree.MustCompilePath(`//`)
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

								switch ext := strings.ToLower(filepath.Ext(inpart.Name)); ext {
								case `.xml`:
									var doc = etree.NewDocument()

									var _, err = doc.ReadFrom(src)
									src.Close()

									if err == nil {
										var multiFirst *etree.Element
										var tmpl string

										for _, el := range doc.Root().FindElementsPath(allDescendants) {
											var txt = el.Text()
											var flushTmpl bool

											if multiFirst == nil { // initializers
												// aka: "index of the first occurrence of '{{' is >= 0 AND occurs before the last occurrence of '}}'"
												if open := strings.Index(txt, `{{`); open >= 0 && open < strings.LastIndex(txt, `}}`) {
													multiFirst = el
													tmpl = txt
													flushTmpl = true
												} else if strings.Contains(txt, `{{`) {
													multiFirst = el
													tmpl = txt
												}
											} else {
												if strings.LastIndex(txt, `{{`) < strings.LastIndex(txt, `}}`) { // terminator
													tmpl += txt
													flushTmpl = true
												} else { // intermediary
													tmpl += txt
												}

												el.SetText(``)
											}

											if flushTmpl && multiFirst != nil {
												if eval, err := EvalInline(
													tmpl,
													options.Data,
													options.FunctionSet,
												); err == nil {
													multiFirst.SetText(eval)
													multiFirst = nil
													tmpl = ``
												} else {
													return fmt.Errorf("bad template %q: %v", tmpl, err)
												}
											}
										}
									} else {
										return fmt.Errorf("bad xml: %v", err)
									}

									_, terr = doc.WriteTo(dest)
								default:
									_, terr = io.Copy(dest, src)
								}

								if terr == nil {
									continue
								} else {
									return terr
								}

								// try to work out a renderer for the file
								// if r, err := GetRenderer(``, self.server); err == nil && doRender {
								// 	var intercept = httptest.NewRecorder()
								// 	var subrender = options

								// 	subrender.Input = src
								// 	subrender.MimeType = inputType
								// 	subrender.RequestedPath = filepath.Join(options.RequestedPath, inpart.Name)

								// 	// render the file into the response interceptor
								// 	if err := r.Render(intercept, req, subrender); err == nil {
								// 		// the result sitting in the interceptor is what we want in the outgoing zip
								// 		if res := intercept.Result(); res != nil && res.Body != nil {
								// 			if _, err := io.Copy(dest, res.Body); err == nil {
								// 				src.Close()
								// 				continue
								// 			} else {
								// 				return err
								// 			}
								// 		} else {
								// 			return fmt.Errorf("empty result from %q", inpart.Name)
								// 		}
								// 	} else {
								// 		return fmt.Errorf("render failed: %v", err)
								// 	}
								// } else if _, err := io.Copy(dest, src); err == nil {
								// 	continue
								// } else {
								// 	return fmt.Errorf("write output: %v", err)
								// }
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
