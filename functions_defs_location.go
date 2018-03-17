package diecast

import "time"

func loadStandardFunctionsLocation(rv FuncMap) {
	// fn sunrise: Return the time of apparent sunrise at the given coordinates, optionally for a given time.
	rv[`sunrise`] = func(latitude float64, longitude float64, atTime ...interface{}) (time.Time, error) {
		sr, _, err := getSunriseSunset(latitude, longitude, atTime...)
		return sr, err
	}

	// fn sunset: Return the time of apparent sunset at the given coordinates, optionally for a given time.
	rv[`sunset`] = func(latitude float64, longitude float64, atTime ...interface{}) (time.Time, error) {
		_, ss, err := getSunriseSunset(latitude, longitude, atTime...)
		return ss, err
	}
}
