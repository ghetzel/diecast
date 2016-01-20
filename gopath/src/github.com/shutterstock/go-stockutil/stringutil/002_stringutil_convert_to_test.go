package stringutil

import (
    "testing"
    "time"
)

func TestConvertToFloat(t *testing.T) {
    if v, err := ConvertTo(Float, "1.5"); err == nil {
        switch v.(type) {
        case float64:
            if v.(float64) != 1.5 {
                t.Errorf("Conversion yielded an incorrect result value: expected 1.5, got: %f", v.(float64))
            }
        default:
            t.Errorf("Conversion yielded an incorrect result type: expected float64, got: %T", v)
        }
    }else{
        t.Errorf("Error during conversion: %v", err)
    }


    if v, err := ConvertTo(Float, "1"); err == nil {
        switch v.(type) {
        case float64:
            if v.(float64) != 1.0 {
                t.Errorf("Conversion yielded an incorrect result value: expected 1.0, got: %f", v.(float64))
            }
        default:
            t.Errorf("Conversion yielded an incorrect result type: expected float64, got: %T", v)
        }
    }else{
        t.Errorf("Error during conversion: %v", err)
    }

    if v, err := ConvertToFloat("1.5"); err == nil {
        if v != float64(1.5) {
            t.Errorf("Conversion yielded an incorrect result value: expected 1.5, got: %f", v)
        }
    }else{
        t.Errorf("Error during conversion: %v", err)
    }

    if v, err := ConvertToFloat("1"); err == nil {
        if v != float64(1.0) {
            t.Errorf("Conversion yielded an incorrect result value: expected 1.0, got: %f", v)
        }
    }else{
        t.Errorf("Error during conversion: %v", err)
    }

    for _, fail := range []string{ `potato`, `true`, `2015-05-01 00:15:16` } {
        if _, err := ConvertTo(Float, fail); err == nil {
            t.Errorf("Conversion should have failed for value '%s', but didn't", fail)
        }

        if _, err := ConvertToFloat(fail); err == nil {
            t.Errorf("Conversion should have failed for value '%s', but didn't", fail)
        }
    }
}

func TestConvertToInteger(t *testing.T) {
    if v, err := ConvertTo(Integer, "7"); err == nil {
        switch v.(type) {
        case int64:
            if v.(int64) != int64(7) {
                t.Errorf("Conversion yielded an incorrect result value: expected 7, got: %f", v.(int64))
            }
        default:
            t.Errorf("Conversion yielded an incorrect result type: expected int64, got: %T", v)
        }
    }else{
        t.Errorf("Error during conversion: %v", err)
    }

    if v, err := ConvertToInteger("7"); err == nil {
        if v != int64(7) {
            t.Errorf("Conversion yielded an incorrect result value: expected 7, got: %f", v)
        }
    }else{
        t.Errorf("Error during conversion: %v", err)
    }

    for _, fail := range []string{ `0.0`, `1.5`, `potato`, `true`, `2015-05-01 00:15:16` } {
        if _, err := ConvertTo(Integer, fail); err == nil {
            t.Errorf("Conversion should have failed for value '%s', but didn't", fail)
        }

        if _, err := ConvertToInteger(fail); err == nil {
            t.Errorf("Conversion should have failed for value '%s', but didn't", fail)
        }
    }
}


func TestConvertToBoolean(t *testing.T) {
    if v, err := ConvertTo(Boolean, "true"); err == nil {
        switch v.(type) {
        case bool:
            if v.(bool) != true {
                t.Errorf("Conversion yielded an incorrect result value: expected true, got: %s", v.(bool))
            }
        default:
            t.Errorf("Conversion yielded an incorrect result type: expected bool, got: %T", v)
        }
    }else{
        t.Errorf("Error during conversion: %v", err)
    }

    if v, err := ConvertTo(Boolean, "false"); err == nil {
        switch v.(type) {
        case bool:
            if v.(bool) != false {
                t.Errorf("Conversion yielded an incorrect result value: expected false, got: %s", v.(bool))
            }
        default:
            t.Errorf("Conversion yielded an incorrect result type: expected bool, got: %T", v)
        }
    }else{
        t.Errorf("Error during conversion: %v", err)
    }

    if v, err := ConvertToBool("true"); err == nil {
        if v != true {
            t.Errorf("Conversion yielded an incorrect result value: expected true, got: %s", v)
        }
    }else{
        t.Errorf("Error during conversion: %v", err)
    }

    if v, err := ConvertToBool("false"); err == nil {
        if v != false {
            t.Errorf("Conversion yielded an incorrect result value: expected false, got: %s", v)
        }
    }else{
        t.Errorf("Error during conversion: %v", err)
    }

    for _, fail := range []string{ `1.5`, `potato`, `1`, `2015-05-01 00:15:16` } {
        if _, err := ConvertTo(Boolean, fail); err == nil {
            t.Errorf("Conversion should have failed for value '%s', but didn't", fail)
        }

        if _, err := ConvertToBool(fail); err == nil {
            t.Errorf("Conversion should have failed for value '%s', but didn't", fail)
        }
    }
}


func TestConvertToDate(t *testing.T) {
    values := map[string]time.Time{
        `2015-05-01 00:15:16`:            time.Date(2015, 5, 1, 0, 15, 16, 0, time.UTC),
        `Fri May 1 00:15:16 UTC 2015`:    time.Date(2015, 5, 1, 0, 15, 16, 0, time.UTC),
        // `Fri May 01 00:15:16 +0000 2015`: time.Date(2015, 5, 1, 0, 15, 16, 0, time.UTC),
        // `01 May 15 00:15 UTC`:            time.Date(2015, 5, 1, 0, 15, 16, 0, time.UTC),
        // `01 May 15 00:15 +0000`:          time.Date(2015, 5, 1, 0, 15, 16, 0, time.UTC),
        // `Friday, 01-May-15 00:15:16 UTC`: time.Date(2015, 5, 1, 0, 15, 16, 0, time.UTC),
    }

    for in, out := range values {
        if v, err := ConvertTo(Time, in); err == nil {
            switch v.(type) {
            case time.Time:
                if v.(time.Time) != out {
                    t.Errorf("Conversion yielded an incorrect result value from '%s': expected %s, got: %s", in, out, v.(time.Time))
                }
            default:
                t.Errorf("Conversion yielded an incorrect result type: expected time.Time, got: %T", v)
            }
        }else{
            t.Errorf("Error during conversion: %v", err)
        }

        if v, err := ConvertToTime(in); err == nil {
            if v != out {
                t.Errorf("Conversion yielded an incorrect result value from '%s': expected %s, got: %f", in, out, v)
            }
        }else{
            t.Errorf("Error during conversion: %v", err)
        }
    }

    for _, fail := range []string{ `1.5`, `potato`, `1`, `false` } {
        if _, err := ConvertTo(Time, fail); err == nil {
            t.Errorf("Conversion should have failed for value '%s', but didn't", fail)
        }

        if _, err := ConvertToTime(fail); err == nil {
            t.Errorf("Conversion should have failed for value '%s', but didn't", fail)
        }
    }
}
