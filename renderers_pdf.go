package diecast

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/ghetzel/go-stockutil/log"
	webfriend "github.com/ghetzel/go-webfriend"
	"github.com/ghetzel/go-webfriend/browser"
	wfcore "github.com/ghetzel/go-webfriend/commands/core"
	"github.com/ghetzel/go-webfriend/commands/page"
)

type PdfRenderer struct {
	server *Server
}

func (self *PdfRenderer) Render(w http.ResponseWriter, req *http.Request, options RenderOptions) error {
	defer options.Input.Close()

	// create a tmp file to write the input to
	if tmp, err := ioutil.TempFile(`diecast`, ``); err == nil {
		defer func() {
			tmp.Close()
			os.Remove(tmp.Name())
		}()

		if _, err := io.Copy(tmp, options.Input); err == nil {
			if _, err := tmp.Seek(0, 0); err != nil {
				return fmt.Errorf("failed to seek intermediate render file")
			}
		} else {
			return fmt.Errorf("failed to write intermediate render file")
		}

		if www, err := browser.Start(); err == nil {
			defer www.Stop()
			var buffer bytes.Buffer

			env := webfriend.NewEnvironment(www)

			if abs, err := filepath.Abs(tmp.Name()); err == nil {
				tmpfile := fmt.Sprintf("file://%v", abs)
				log.Debugf("Rendering %v as PDF", tmpfile)

				if m, ok := env.Module(`core`); ok {
					if core, ok := m.(*wfcore.Commands); ok {
						if _, err := core.Go(tmpfile, &wfcore.GoArgs{
							RequireOriginatingRequest: false,
						}); err != nil {
							return err
						}
					} else {
						return fmt.Errorf("Unable to retrieve Webfriend Core module")
					}
				} else {
					return fmt.Errorf("Unable to retrieve Webfriend Core module")
				}

				if m, ok := env.Module(`page`); ok {
					if page, ok := m.(*page.Commands); ok {
						if err := page.Pdf(&buffer, nil); err == nil {
							w.Header().Set(`Content-Type`, `application/pdf`)
							_, err := io.Copy(w, &buffer)

							return err
						} else {
							return err
						}
					} else {
						return fmt.Errorf("Unable to retrieve Webfriend Page module")
					}
				} else {
					return fmt.Errorf("Unable to retrieve Webfriend Page module")
				}
			} else {
				return fmt.Errorf("failed to get intermediate filename: %v", err)
			}
		} else {
			log.Fatalf("could not generate PDF: %v", err)
			return err
		}
	} else {
		return err
	}
}
