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
	rv[`contains`] = strings.Contains
	rv[`lower`] = strings.ToLower
	rv[`ltrim`] = strings.TrimPrefix
	rv[`replace`] = strings.Replace
	rv[`rtrim`] = strings.TrimSuffix
	rv[`split`] = strings.Split
	rv[`splitn`] = strings.SplitN
	rv[`strcount`] = strings.Count
	rv[`titleize`] = strings.ToTitle
	rv[`trim`] = strings.TrimSpace
	rv[`upper`] = strings.ToUpper

	// encoding
	rv[`jsonify`] = func(value interface{}) (string, error) {
		data, err := json.Marshal(value)
		return string(data[:]), err
	}

	// type handling and conversion
	rv[`is_bool`] = stringutil.IsBoolean
	rv[`is_int`] = stringutil.IsInteger
	rv[`is_float`] = stringutil.IsFloat
	rv[`autotype`] = stringutil.Autotype
	rv[`as_str`] = stringutil.ToString
	rv[`as_int`] = stringutil.ConvertToInteger
	rv[`as_float`] = stringutil.ConvertToFloat
	rv[`as_bool`] = stringutil.ConvertToBool
	rv[`as_time`] = stringutil.ConvertToTime

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

	rv[`time`] = tmFmt
	rv[`now`] = func(format ...string) (string, error) {
		return tmFmt(time.Now(), format...)
	}

	return rv
}
