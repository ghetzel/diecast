package stringutil

import (
    "fmt"
    "math"
    "reflect"
    "strconv"
    "strings"
    "time"
)

type SiPrefix int
const (
    None SiPrefix = 0
    Kilo          = 1
    Mega          = 2
    Giga          = 3
    Tera          = 4
    Peta          = 5
    Exa           = 6
    Zetta         = 7
    Yotta         = 8
)

type ConvertType int
const (
    String ConvertType = 0
    Boolean            = 1
    Float              = 2
    Integer            = 3
    Time               = 4
)

var DateConvertFormats = []string{
    "2006-01-02 15:04:05",
    "1136239445",
    time.RFC3339,
    time.ANSIC,
    time.UnixDate,
    time.RubyDate,
    time.RFC822,
    time.RFC822Z,
    time.RFC1123,
    time.RFC1123Z,
    time.Kitchen,
}

func IsInteger(in string) bool {
    if _, err := strconv.Atoi(in); err == nil {
        return true
    }

    return false
}


func IsFloat(in string) bool {
    if _, err := strconv.ParseFloat(in, 64); err == nil {
        return true
    }

    return false
}


func IsBoolean(in string) bool {
    if in == `true` || in == `false` {
        return true
    }

    return false
}


func ToString(in interface{}) (string, error) {
    switch reflect.TypeOf(in).Kind() {
    case reflect.Int:
        return strconv.FormatInt(int64(in.(int)), 10), nil
    case reflect.Int8:
        return strconv.FormatInt(int64(in.(int8)), 10), nil
    case reflect.Int16:
        return strconv.FormatInt(int64(in.(int16)), 10), nil
    case reflect.Int32:
        return strconv.FormatInt(int64(in.(int32)), 10), nil
    case reflect.Int64:
        return strconv.FormatInt(in.(int64), 10), nil
    case reflect.Uint:
        return strconv.FormatUint(uint64(in.(uint)), 10), nil
    case reflect.Uint8:
        return strconv.FormatUint(uint64(in.(uint8)), 10), nil
    case reflect.Uint16:
        return strconv.FormatUint(uint64(in.(uint16)), 10), nil
    case reflect.Uint32:
        return strconv.FormatUint(uint64(in.(uint32)), 10), nil
    case reflect.Uint64:
        return strconv.FormatUint(in.(uint64), 10), nil
    case reflect.Float32:
        return strconv.FormatFloat(float64(in.(float32)), 'f', -1, 32), nil
    case reflect.Float64:
        return strconv.FormatFloat(in.(float64), 'f', -1, 64), nil
    case reflect.Bool:
        return strconv.FormatBool(in.(bool)), nil
    case reflect.String:
        return in.(string), nil
    default:
        return "", fmt.Errorf("Unable to convert type '%T' to string", in)
    }
}

func GetSiPrefix(input string) (SiPrefix, error) {
    switch input {
    case "", "b", "B":
        return None, nil
    case "k", "K":
        return Kilo, nil
    case "m", "M":
         return Mega, nil
    case "g", "G":
        return Giga, nil
    case "t", "T":
        return Tera, nil
    case "p", "P":
        return Peta, nil
    case "e", "E":
        return Exa, nil
    case "z", "Z":
        return Zetta, nil
    case "y", "Y":
        return Yotta, nil
    default:
        return None, fmt.Errorf("Unrecognized SI unit '%s'", input)
    }
}


func ToBytes(input string) (float64, error) {
//  handle -ibibyte suffixes like KiB, GiB
    if strings.HasSuffix(input, "ib") || strings.HasSuffix(input, "iB") {
        input = input[0:len(input)-2]

//  handle input that puts the 'B' in the suffix; e.g.: Kb, GB
    }else if len(input) > 2 && IsInteger(string(input[len(input)-3])) && (input[len(input)-1] == 'b' || input[len(input)-1] == 'B') {
        input = input[0:len(input)-1]
    }

    if prefix, err := GetSiPrefix(string(input[len(input)-1])); err == nil {
        if v, err := strconv.ParseFloat(input[0:len(input)-1], 64); err == nil {
            return v * math.Pow(1024, float64(prefix)), nil
        }else{
            return 0, err
        }
    }else{
        if v, err := strconv.ParseFloat(input, 64); err == nil {
            return v, nil
        }else{
            return 0, fmt.Errorf("Unrecognized input string '%s'", input)
        }
    }
}

func ConvertTo(toType ConvertType, in string) (interface{}, error) {
    switch toType {
    case Float:
        return strconv.ParseFloat(in, 64)
    case Integer:
        return strconv.ParseInt(in, 10, 64)
    case Boolean:
        if IsBoolean(in) {
            return (in == `true`), nil
        }else{
            return nil, fmt.Errorf("Cannot convert '%s' into a boolean value", in)
        }
    case Time:
        for _, format := range DateConvertFormats {
            if tm, err := time.Parse(format, in); err == nil {
                return tm, nil
            }
        }

        return nil, fmt.Errorf("Cannot convert '%s' into a date/time value", in)

    default:
        return in, nil
    }
}

func ConvertToInteger(in string) (int64, error) {
    if v, err := ConvertTo(Integer, in); err == nil {
        return v.(int64), nil
    }else{
        return int64(0), err
    }
}

func ConvertToFloat(in string) (float64, error) {
    if v, err := ConvertTo(Float, in); err == nil {
        return v.(float64), nil
    }else{
        return float64(0.0), err
    }
}

func ConvertToString(in string) (string, error) {
    if v, err := ConvertTo(String, in); err == nil {
        return v.(string), nil
    }else{
        return ``, err
    }
}

func ConvertToBool(in string) (bool, error) {
    if v, err := ConvertTo(Boolean, in); err == nil {
        return v.(bool), nil
    }else{
        return false, err
    }
}

func ConvertToTime(in string) (time.Time, error) {
    if v, err := ConvertTo(Time, in); err == nil {
        return v.(time.Time), nil
    }else{
        return time.Time{}, err
    }
}
