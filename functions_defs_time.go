package diecast

import (
	"fmt"
	"strings"
	"time"

	"github.com/ghetzel/go-stockutil/stringutil"
	"github.com/ghetzel/go-stockutil/timeutil"
)

func loadStandardFunctionsTime(rv FuncMap) {
	// fn time: Return the given Time formatted using *format*.  See [Time Formats](#time-formats) for
	//          acceptable formats.
	rv[`time`] = tmFmt

	// fn now: Return the current time formatted using *format*.  See [Time Formats](#time-formats) for
	//          acceptable formats.
	rv[`now`] = func(format ...string) (string, error) {
		return tmFmt(time.Now(), format...)
	}

	// fn addTime: Return a time with with given *duration* added to it.  Can specify time *at* to
	//             apply the change to.
	rv[`addTime`] = func(durationString string, atI ...interface{}) (time.Time, error) {
		at := time.Now()

		if len(atI) > 0 {
			if tm, err := stringutil.ConvertToTime(atI[0]); err == nil {
				at = tm
			} else {
				return time.Time{}, err
			}
		}

		if duration, err := timeutil.ParseDuration(durationString); err == nil {
			return at.Add(duration), nil
		} else {
			return time.Time{}, err
		}
	}

	// fn ago: Return a Time subtracted by the given *duration*.
	rv[`ago`] = func(durationString string, fromTime ...time.Time) (time.Time, error) {
		from := time.Now()

		if len(fromTime) > 0 {
			from = fromTime[0]
		}

		if duration, err := timeutil.ParseDuration(durationString); err == nil {
			return from.Add(-1 * duration), nil
		} else {
			return time.Time{}, err
		}
	}

	// fn since: Return the amount of time that has elapsed since *time*, optionally rounded
	//           to the nearest *interval*.
	rv[`since`] = func(at interface{}, interval ...string) (time.Duration, error) {
		if tm, err := stringutil.ConvertToTime(at); err == nil {
			since := time.Since(tm)

			if len(interval) > 0 {
				switch strings.ToLower(interval[0]) {
				case `s`, `sec`, `second`:
					since = since.Round(time.Second)
				case `m`, `min`, `minute`:
					since = since.Round(time.Minute)
				case `h`, `hr`, `hour`:
					since = since.Round(time.Hour)
				}
			}

			return since, nil
		} else {
			return 0, err
		}
	}

	// fn duration: Convert the given *value* from a duration of *unit* into the given time *format*.
	rv[`duration`] = func(value interface{}, unit string, formats ...string) (string, error) {
		if v, err := stringutil.ConvertToInteger(value); err == nil {
			duration := time.Duration(v)
			format := `timer`

			if len(formats) > 0 {
				format = formats[0]
			}

			switch unit {
			case `ns`, ``:
				break
			case `us`:
				duration = duration * time.Microsecond
			case `ms`:
				duration = duration * time.Millisecond
			case `s`:
				duration = duration * time.Second
			case `m`:
				duration = duration * time.Minute
			case `h`:
				duration = duration * time.Hour
			case `d`:
				duration = duration * time.Hour * 24
			case `y`:
				duration = duration * time.Hour * 24 * 365
			default:
				return ``, fmt.Errorf("Unrecognized unit %q", unit)
			}

			basetime := time.Date(0, 0, 0, 0, 0, 0, 0, time.UTC)
			basetime = basetime.Add(duration)

			return tmFmt(basetime, format)
		} else {
			return ``, err
		}
	}

	// fn isBefore: Return whether the *first* time is before the *second* one.
	rv[`isBefore`] = func(first interface{}, secondI ...interface{}) (bool, error) {
		return timeCmp(true, first, secondI...)
	}

	// fn isAfter: Return whether the *first* time is after the *second* one.
	rv[`isAfter`] = func(first interface{}, secondI ...interface{}) (bool, error) {
		return timeCmp(false, first, secondI...)
	}
}
