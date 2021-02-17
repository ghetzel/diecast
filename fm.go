package diecast

import (
	"bytes"
	"io"
	"net/http"

	"gopkg.in/yaml.v2"
)

type FrontMatter struct {
	DataSources   DataSet       `yaml:"data"`
	ContentOffset int           `yaml:"-"`
	Request       *http.Request `yaml:"-"`
}

// Parses an input ReadCloser, splitting out the Front Matter and body of a templated file.
// Files that do not contain a Front Matter section will return a nil *FrontMatter.
func SplitFrontMatter(source io.Reader) (io.Reader, *FrontMatter, error) {
	if source == nil {
		return nil, nil, io.EOF
	}

	var chunk = make([]byte, len(FrontMatterSeparator))

	// Front Matter is declared in the first 4 bytes of the file being `---\n`.  If this is not the case,
	// then we kinda glue that first 4 bytes back in place and return an intact, (effectively) unread io.Reader.
	if n, err := io.ReadFull(source, chunk); err == nil {
		if bytes.Equal(chunk, FrontMatterSeparator) {
			var fmData = make([]byte, 0, MaxFrontMatterSize)

			// Limited byte-by-byte read, checking for the FrontMatterSeparator acting as a terminator.
			for i := 0; i < MaxFrontMatterSize; i++ {
				var b = make([]byte, 1)

				if _, err := source.Read(b); err == nil || err == io.EOF {
					fmData = append(fmData, b...)

					if len(fmData) >= 4 {
						var back4 = len(fmData) - 4

						if last4 := fmData[back4:]; bytes.Equal(last4, FrontMatterSeparator) {
							fmData = fmData[:back4]
							break
						}
					}
				} else {
					return source, nil, err
				}
			}

			var fm = FrontMatter{
				ContentOffset: len(FrontMatterSeparator) + len(fmData),
			}

			// Only attempt to parse if we actually read any front matter data.
			if len(fmData) > 0 {

				if err := yaml.UnmarshalStrict(fmData, &fm); err == nil {
					return source, &fm, nil
				} else {
					return source, nil, err
				}
			} else {
				return source, &fm, nil
			}
		} else {
			// paste the bit we just read pack onto the front of the io.Reader like nothing happened
			return io.MultiReader(bytes.NewBuffer(chunk[0:n]), source), nil, nil
		}
	} else {
		return source, nil, err
	}
}
