package diecast

import (
	"encoding/json"
	"fmt"
	"github.com/ghetzel/go-stockutil/stringutil"
	"html/template"
	"strings"
	"time"
)

func GetStandardFunctions() template.FuncMap {
	rv := make(template.FuncMap)

	// string processing
	rv[`Contains`] = strings.Contains
	rv[`Lower`] = strings.ToLower
	rv[`TrimPrefix`] = strings.TrimPrefix
	rv[`Replace`] = strings.Replace
	rv[`TrimSuffix`] = strings.TrimSuffix
	rv[`Split`] = strings.Split
	rv[`SplitN`] = strings.SplitN
	rv[`StrCount`] = strings.Count
	rv[`Titleize`] = strings.ToTitle
	rv[`Trim`] = strings.TrimSpace
	rv[`Upper`] = strings.ToUpper
	rv[`HasPrefix`] = strings.HasPrefix
	rv[`HasSuffix`] = strings.HasSuffix

	// encoding
	rv[`Jsonify`] = func(value interface{}, indent ...string) (string, error) {
		indentString := ``

		if len(indent) > 0 {
			indentString = indent[0]
		}

		data, err := json.MarshalIndent(value, ``, indentString)
		return string(data[:]), err
	}

	// type handling and conversion
	rv[`IsBool`] = stringutil.IsBoolean
	rv[`IsInt`] = stringutil.IsInteger
	rv[`IsFloat`] = stringutil.IsFloat
	rv[`Autotype`] = stringutil.Autotype
	rv[`AsStr`] = stringutil.ToString
	rv[`AsInt`] = stringutil.ConvertToInteger
	rv[`AsFloat`] = stringutil.ConvertToFloat
	rv[`AsBool`] = stringutil.ConvertToBool
	rv[`AsTime`] = stringutil.ConvertToTime

	// time and date formatting
	tmFmt := func(value interface{}, format ...string) (string, error) {
		if v, err := stringutil.ConvertToTime(value); err == nil {
			var tmFormat string

			if len(format) == 0 {
				tmFormat = time.RFC3339
			} else {
				switch format[0] {
				case `kitchen`:
					tmFormat = time.Kitchen
				case `rfc3339`:
					tmFormat = time.RFC3339
				case `rfc3339ns`:
					tmFormat = time.RFC3339Nano
				case `rfc822`:
					tmFormat = time.RFC822
				case `rfc822z`:
					tmFormat = time.RFC822Z
				case `epoch`:
					return fmt.Sprintf("%d", v.Unix()), nil
				case `epoch-ms`:
					return fmt.Sprintf("%d", int64(v.UnixNano()/1000000)), nil
				case `epoch-us`:
					return fmt.Sprintf("%d", int64(v.UnixNano()/1000)), nil
				case `day`:
					tmFormat = `Monday`
				case `slash`:
					tmFormat = `01/02/2006`
				case `slash-dmy`:
					tmFormat = `02/01/2006`
				case `ymd`:
					tmFormat = `2006-01-02`
				case `ruby`:
					tmFormat = time.RubyDate
				default:
					tmFormat = format[0]
				}
			}

			return v.Format(tmFormat), nil
		} else {
			return ``, err
		}
	}

	rv[`Time`] = tmFmt
	rv[`Now`] = func(format ...string) (string, error) {
		return tmFmt(time.Now(), format...)
	}

	return rv
}
