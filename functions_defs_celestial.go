package diecast

import (
	"time"

	"github.com/ghetzel/go-stockutil/geoutil"
	"github.com/ghetzel/go-stockutil/maputil"
	"github.com/ghetzel/go-stockutil/mathutil"
	"github.com/ghetzel/go-stockutil/typeutil"
	"github.com/sj14/astral"
)

const lunarHalfCycle = 14
const sunElevationDayNightCutoff = -0.523 // day starts/ends when the sun is 0.7deg below the horizon

func loadStandardFunctionsCelestial(funcs FuncMap, server *Server) funcGroup {
	return funcGroup{
		Name:        `Celestial & Astronomical Functions`,
		Description: `Used for calculating details pertaining to the motion of celestial bodies as viewed from points on Earth.`,
		Functions: []funcDef{
			{
				Name:    `celestial`,
				Summary: `Return extensive details about the position, timing, and motion of celestial objects at a given time and location.`,
				Arguments: []funcArg{
					{
						Name:        `time`,
						Type:        `time`,
						Description: `The date/time of the observation.`,
						Default:     `t`,
					}, {
						Name:        `latitude`,
						Type:        `number`,
						Description: `The latitude of the observation point.`,
					}, {
						Name:        `longitude`,
						Type:        `number`,
						Description: `The longitude of the observation point.`,
					}, {
						Name:        `elevation`,
						Type:        `number`,
						Description: `The elevation the observation point.`,
						Optional:    true,
					},
				},
				Examples: []funcExample{
					{
						Code: `celestial "2021-06-29T22:45:42-04:00" 40.698828 -75.866871`,
						Return: map[string]interface{}{
							"observer": map[string]interface{}{
								"elevation": 0,
								"latitude":  40.698828,
								"longitude": -75.866871,
								"time":      "2021-06-29T22:45:42-04:00",
							},
							"sun": map[string]interface{}{
								"dawn": map[string]interface{}{
									"astronomical": "2021-06-29T03:30:13-04:00",
									"blue_hour": map[string]interface{}{
										"end":   "2021-06-29T05:15:06-04:00",
										"start": "2021-06-29T05:02:08-04:00",
									},
									"civil": "2021-06-29T05:02:08-04:00",
									"golden_hour": map[string]interface{}{
										"end":   "2021-06-29T06:16:06-04:00",
										"start": "2021-06-29T05:15:06-04:00",
									},
									"nautical": "2021-06-29T04:19:57-04:00",
								},
								"daytime": map[string]interface{}{
									"end":            "2021-06-29T20:38:18-04:00",
									"length_minutes": 902,
									"start":          "2021-06-29T05:36:01-04:00",
								},
								"dusk": map[string]interface{}{
									"astronomical": "2021-06-29T22:43:51-04:00",
									"blue_hour": map[string]interface{}{
										"end":   "2021-06-29T21:12:09-04:00",
										"start": "2021-06-29T20:59:11-04:00",
									},
									"civil": "2021-06-29T21:12:09-04:00",
									"golden_hour": map[string]interface{}{
										"end":   "2021-06-29T20:59:11-04:00",
										"start": "2021-06-29T19:58:15-04:00",
									},
									"nautical": "2021-06-29T21:54:15-04:00",
								},
								"midnight": "2021-06-29T01:07:14-04:00",
								"night": map[string]interface{}{
									"end":            "2021-06-30T05:02:40-04:00",
									"length_minutes": 471,
									"start":          "2021-06-29T21:12:09-04:00",
								},
								"noon": "2021-06-29T13:07:06-04:00",
								"position": map[string]interface{}{
									"azimuth": map[string]interface{}{
										"angle":    108.1,
										"cardinal": "E",
									},
									"elevation": map[string]interface{}{
										"angle":           -18.104,
										"refracted_angle": -18.086,
									},
									"zenith": map[string]interface{}{
										"angle":           325.8,
										"cardinal":        "NW",
										"refracted_angle": 325.8,
									},
								},
								"state": map[string]interface{}{
									"blue_hour":   false,
									"daytime":     false,
									"golden_hour": false,
									"night":       true,
									"twilight":    false,
								},
								"sunrise": "2021-06-29T05:36:01-04:00",
								"sunset":  "2021-06-29T20:38:18-04:00",
								"twilight": map[string]interface{}{
									"end":   "2021-06-29T21:12:09-04:00",
									"start": "2021-06-29T20:38:18-04:00",
								},
							},
						},
					},
				},
				Function: func(obstime interface{}, latitude interface{}, longitude interface{}, e ...interface{}) (map[string]interface{}, error) {
					var out = maputil.M(nil)
					var now = time.Now()
					var t = typeutil.OrTime(obstime, now).Round(time.Second)
					var elevation = typeutil.OrFloat(e)
					var o = astral.Observer{
						Latitude:  typeutil.Float(latitude),
						Longitude: typeutil.Float(longitude),
						Elevation: elevation,
					}

					var sunRefractedElevation = astral.Elevation(o, t, true)
					var azimuth, zenithT = astral.ZenithAndAzimuth(o, t, false)
					var _, zenithR = astral.ZenithAndAzimuth(o, t, true)
					var stateName string
					var dawnBH bool
					var dawnGH bool
					var duskGH bool
					var duskBH bool

					// all those enchanting sounding parts of the sun's journey through the sky...
					var dawnA, _ = astral.Dawn(o, t, astral.DepressionAstronomical)
					var dawnN, _ = astral.Dawn(o, t, astral.DepressionNautical)
					var dawnC, _ = astral.Dawn(o, t, astral.DepressionCivil)
					var sunrise time.Time
					var noon = astral.Noon(o, t).Round(time.Second)
					var sunset time.Time
					var duskC, _ = astral.Dusk(o, t, astral.DepressionCivil)
					var duskN, _ = astral.Dusk(o, t, astral.DepressionNautical)
					var duskA, _ = astral.Dusk(o, t, astral.DepressionAstronomical)
					var midnight = astral.Midnight(o, t).Round(time.Second)

					out.Set(`observer.time`, t.Format(time.RFC3339))
					out.Set(`observer.latitude`, o.Latitude)
					out.Set(`observer.longitude`, o.Longitude)
					out.Set(`observer.elevation`, o.Elevation)

					out.Set(`sun.state.blue_hour`, false)
					out.Set(`sun.state.golden_hour`, false)
					out.Set(`sun.state.twilight`, false)
					out.Set(`sun.state.daytime`, (sunRefractedElevation >= sunElevationDayNightCutoff))
					out.Set(`sun.state.night`, (sunRefractedElevation < sunElevationDayNightCutoff))
					out.Set(`sun.state.golden_hour`, false)
					out.Set(`sun.state.twilight`, false)
					out.Set(`sun.state.blue_hour`, false)
					out.Set(`sun.position.zenith.angle`, mathutil.RoundPlaces(zenithT, 2))
					out.Set(`sun.position.zenith.refracted_angle`, mathutil.RoundPlaces(zenithR, 2))
					out.Set(`sun.position.zenith.cardinal`, geoutil.GetDirectionFromBearing(zenithT))
					out.Set(`sun.position.azimuth.angle`, mathutil.RoundPlaces(azimuth, 2))
					out.Set(`sun.position.azimuth.cardinal`, geoutil.GetDirectionFromBearing(azimuth))
					out.Set(`sun.position.elevation.angle`, mathutil.RoundPlaces(astral.Elevation(o, t, false), 3))
					out.Set(`sun.position.elevation.refracted_angle`, mathutil.RoundPlaces(sunRefractedElevation, 3))
					out.Set(`sun.dawn.astronomical`, dawnA.Round(time.Second))
					out.Set(`sun.dawn.nautical`, dawnN.Round(time.Second))
					out.Set(`sun.dawn.civil`, dawnC.Round(time.Second))

					// ---------------------------------------------------------------------------------------
					if start, end, err := astral.BlueHour(o, t, astral.SunDirectionRising); err == nil {
						out.Set(`sun.dawn.blue_hour.start`, start.Round(time.Second))
						out.Set(`sun.dawn.blue_hour.end`, end.Round(time.Second))

						if t.After(start) && t.Before(end) {
							dawnBH = true
							out.Set(`sun.state.blue_hour`, dawnBH)
						}
					} else {
						return nil, err
					}

					// ---------------------------------------------------------------------------------------
					if start, end, err := astral.GoldenHour(o, t, astral.SunDirectionRising); err == nil {
						out.Set(`sun.dawn.golden_hour.start`, start.Round(time.Second))
						out.Set(`sun.dawn.golden_hour.end`, end.Round(time.Second))

						if t.After(start) && t.Before(end) {
							dawnGH = true
							out.Set(`sun.state.golden_hour`, dawnGH)
						}
					} else {
						return nil, err
					}

					// ---------------------------------------------------------------------------------------
					if at, err := astral.Sunrise(o, t); err == nil {
						sunrise = at.Round(time.Second)
						out.Set(`sun.sunrise`, sunrise)
					}

					// ---------------------------------------------------------------------------------------
					if start, end, err := astral.Twilight(o, t, astral.SunDirectionRising); err == nil {
						out.Set(`sun.twilight.start`, start.Round(time.Second))
						out.Set(`sun.twilight.end`, end.Round(time.Second))

						if t.After(start) && t.Before(end) {
							out.Set(`sun.state.twilight`, true)
						}
					} else {
						return nil, err
					}

					// ---------------------------------------------------------------------------------------
					if start, end, err := astral.Daylight(o, t); err == nil {
						out.Set(`sun.daytime.start`, start.Round(time.Second))
						out.Set(`sun.daytime.end`, end.Round(time.Second))
						out.Set(`sun.daytime.length_minutes`, int(end.Sub(start).Round(time.Minute)/time.Minute))
					} else {
						return nil, err
					}

					// ---------------------------------------------------------------------------------------
					out.Set(`sun.noon`, noon.Round(time.Second))

					// ---------------------------------------------------------------------------------------
					if start, end, err := astral.GoldenHour(o, t, astral.SunDirectionSetting); err == nil {
						out.Set(`sun.dusk.golden_hour.start`, start.Round(time.Second))
						out.Set(`sun.dusk.golden_hour.end`, end.Round(time.Second))

						if t.After(start) && t.Before(end) {
							duskGH = true
							out.Set(`sun.state.golden_hour`, duskGH)
						}
					} else {
						return nil, err
					}

					// ---------------------------------------------------------------------------------------
					if at, err := astral.Sunset(o, t); err == nil {
						sunset = at.Round(time.Second)
						out.Set(`sun.sunset`, sunset)
					}

					// ---------------------------------------------------------------------------------------
					if start, end, err := astral.Twilight(o, t, astral.SunDirectionSetting); err == nil {
						out.Set(`sun.twilight.start`, start.Round(time.Second))
						out.Set(`sun.twilight.end`, end.Round(time.Second))

						if t.After(start) && t.Before(end) {
							out.Set(`sun.state.twilight`, true)
						}
					} else {
						return nil, err
					}

					// ---------------------------------------------------------------------------------------
					if start, end, err := astral.BlueHour(o, t, astral.SunDirectionSetting); err == nil {
						out.Set(`sun.dusk.blue_hour.start`, start.Round(time.Second))
						out.Set(`sun.dusk.blue_hour.end`, end.Round(time.Second))

						if t.After(start) && t.Before(end) {
							duskBH = true
							out.Set(`sun.state.blue_hour`, duskBH)
						}
					} else {
						return nil, err
					}

					// ---------------------------------------------------------------------------------------

					out.Set(`sun.dusk.civil`, duskC.Round(time.Second))
					out.Set(`sun.dusk.nautical`, duskN.Round(time.Second))
					out.Set(`sun.dusk.astronomical`, duskA.Round(time.Second))

					// ---------------------------------------------------------------------------------------
					if start, end, err := astral.Night(o, t); err == nil {
						out.Set(`sun.night.start`, start.Round(time.Second))
						out.Set(`sun.night.end`, end.Round(time.Second))
						out.Set(`sun.night.length_minutes`, int(end.Sub(start).Round(time.Minute)/time.Minute))
					} else {
						return nil, err
					}

					if t.After(midnight.Add(-15*time.Minute)) && t.Before(midnight.Add(15*time.Minute)) {
						stateName = `solar-midnight`
					} else if duskGH || (t.After(sunset) && t.Before(duskC)) {
						stateName = `sunset`
					} else if t.After(duskA) && t.Before(duskA.Add(15*time.Minute)) {
						stateName = `astronomical-twilight`
					} else if t.After(duskN) && t.Before(duskA) {
						stateName = `nautical-twilight`
					} else if t.After(duskC) && t.Before(duskN) {
						stateName = `civil-twilight`
					} else if t.After(dawnC) && t.Before(sunrise) {
						stateName = `civil-twilight`
					} else if t.After(dawnN) && t.Before(dawnC) {
						stateName = `nautical-twilight`
					} else if t.After(dawnA) && t.Before(dawnN) {
						stateName = `astronomical-twilight`
					} else if dawnGH {
						stateName = `sunrise`
					} else if t.After(noon.Add(-15*time.Minute)) && t.Before(noon.Add(15*time.Minute)) {
						stateName = `solar-noon`
					} else if t.After(sunrise) && t.Before(noon) {
						stateName = `morning`
					} else if t.After(noon) && t.Before(sunset) {
						stateName = `afternoon`
					} else if sunRefractedElevation < sunElevationDayNightCutoff {
						stateName = `night`
					} else if sunRefractedElevation >= sunElevationDayNightCutoff {
						stateName = `day`
					}

					out.Set(`sun.state.name`, stateName)

					// ---------------------------------------------------------------------------------------
					out.Set(`sun.midnight`, midnight)

					// ---------------------------------------------------------------------------------------

					return out.MapNative(), nil
				},
			},
		},
	}
}
