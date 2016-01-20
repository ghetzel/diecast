package maputil

import (
    "fmt"
    "strings"
    "strconv"
    "sort"
    "github.com/shutterstock/go-stockutil/stringutil"
    _ "log"
    "reflect"
)


func StringKeys(input map[string]interface{}) []string {
    keys := make([]string, 0)

    for k, _ := range input {
        keys = append(keys, k)
    }

    return keys
}

func MapValues(input map[string]interface{}) []interface{} {
    values := make([]interface{}, 0)

    for _, value := range input {
        values = append(values, value)
    }

    return values
}

func Join(input map[string]interface{}, innerJoiner string, outerJoiner string) string {
    parts := make([]string, 0)

    for key, value := range input {
        if v, err := stringutil.ToString(value); err == nil {
            parts = append(parts, key + innerJoiner + v)
        }
    }

    return strings.Join(parts, outerJoiner)
}

func Split(input string, innerJoiner string, outerJoiner string) map[string]interface{} {
    rv    := make(map[string]interface{})
    pairs := strings.Split(input, outerJoiner)

    for _, pair := range pairs {
        kv := strings.SplitN(pair, innerJoiner, 2)

        if len(kv) == 2 {
            rv[ kv[0] ] = kv[1]
        }
    }

    return rv
}

// Take a flat (non-nested) map keyed with fields joined on fieldJoiner and return a
// deeply-nested map
//
func DiffuseMap(data map[string]interface{}, fieldJoiner string) (map[string]interface{}, error) {
    rv, _ := DiffuseMapTyped(data, fieldJoiner, "")
    return rv, nil
}


// Take a flat (non-nested) map keyed with fields joined on fieldJoiner and return a
// deeply-nested map
//
func DiffuseMapTyped(data map[string]interface{}, fieldJoiner string, typePrefixSeparator string) (map[string]interface{}, []error) {
    errs   := make([]error, 0)
    output := make(map[string]interface{})

//  get the list of keys and sort them because order in a map is undefined
    dataKeys := StringKeys(data)
    sort.Strings(dataKeys)

//  for each data item
    for _, key := range dataKeys {
        var keyParts []string

        value, _ := data[key]

    //  handle "typed" maps in which the type information is embedded
        if typePrefixSeparator != "" {
            typeKeyPair := strings.SplitN(key, typePrefixSeparator, 2)

            if len(typeKeyPair) == 2 {
                key = typeKeyPair[1]

                if v, err := coerceIntoType(value, typeKeyPair[0]); err == nil {
                    value = v
                }else{
                    errs = append(errs, err)
                }
            }else{
                if v, err := coerceIntoType(value, `str`); err == nil {
                    value = v
                }else{
                    errs = append(errs, err)
                }
            }

        }

        keyParts = strings.Split(key, fieldJoiner)

        output = DeepSet(output, keyParts, value).(map[string]interface{})
    }

    return output, errs
}


// Take a deeply-nested map and return a flat (non-nested) map with keys whose intermediate tiers are joined with fieldJoiner
//
func CoalesceMap(data map[string]interface{}, fieldJoiner string) (map[string]interface{}, error) {
    return deepGetValues([]string{}, fieldJoiner, data), nil
}

// Take a deeply-nested map and return a flat (non-nested) map with keys whose intermediate tiers are joined with fieldJoiner
// Additionally, values will be converted to strings and keys will be prefixed with the datatype of the value
//
func CoalesceMapTyped(data map[string]interface{}, fieldJoiner string, typePrefixSeparator string) (map[string]interface{}, []error) {
    errs := make([]error, 0)
    rv := make(map[string]interface{})

    for k, v := range deepGetValues([]string{}, fieldJoiner, data) {
        if stringVal, err := stringutil.ToString(v); err == nil {
            rv[prepareCoalescedKey(k, v, typePrefixSeparator)] = stringVal
        }else{
            errs = append(errs, err)
        }
    }

    return rv, errs
}


func deepGetValues(keys []string, joiner string, data interface{}) map[string]interface{} {
    rv := make(map[string]interface{})

    if data != nil {
        switch reflect.TypeOf(data).Kind() {
        case reflect.Map:
            for k, v := range data.(map[string]interface{}){
                newKey := keys
                newKey = append(newKey, k)

                for kk, vv := range deepGetValues(newKey, joiner, v) {
                    rv[kk] = vv
                }
            }

        case reflect.Slice, reflect.Array:
            for i, value := range data.([]interface{}) {
                newKey := keys
                newKey = append(newKey, strconv.Itoa(i))

                for k, v := range deepGetValues(newKey, joiner, value){
                    rv[k] = v
                }
            }

        default:
            rv[strings.Join(keys, joiner)] = data
        }
    }

    return rv
}


func prepareCoalescedKey(key string, value interface{}, typePrefixSeparator string) string {
    if typePrefixSeparator == "" {
        return key
    }else{
        var datatype string

        switch reflect.TypeOf(value).Kind() {
        case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
             reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
            datatype = "int"
        case reflect.Float32, reflect.Float64:
            datatype = "float"
        case reflect.Bool:
            datatype = "bool"
        case reflect.String:
            datatype = "str"
        default:
            return key
        }

        return datatype + typePrefixSeparator + key
    }
}


func coerceIntoType(in interface{}, typeName string) (interface{}, error) {
//  make sure `in' is a string, error out if not
    if inStr, err := stringutil.ToString(in); err == nil {
        switch typeName {
        case `bool`:
            if v, err := strconv.ParseBool(inStr); err == nil {
                return interface{}(v), nil
            }else{
                return nil, fmt.Errorf("Unable to convert '%s' into a boolean", inStr)
            }
        case `int`:
            if v, err := strconv.ParseInt(inStr, 10, 64); err == nil {
                return interface{}(v), nil
            }else{
                return nil, fmt.Errorf("Unable to convert '%s' into an integer", inStr)
            }
        case `float`:
            if v, err := strconv.ParseFloat(inStr, 64); err == nil {
                return interface{}(v), nil
            }else{
                return nil, fmt.Errorf("Unable to convert '%s' into a float", inStr)
            }
        case `str`:
            return interface{}(inStr), nil
        default:
            return in, fmt.Errorf("Unknown conversion type '%s'", typeName)
        }

    }else{
        return in, nil
    }
}

func DeepGet(data interface{}, path []string, fallback interface{}) interface{} {
    current := data

    for i := 0; i < len(path); i++ {
        part := path[i]

        switch current.(type) {
    //  arrays
        case []interface{}:
            currentAsArray := current.([]interface{})

            if stringutil.IsInteger(part) {
                if partIndex, err := strconv.Atoi(part); err == nil {
                    if partIndex < len(currentAsArray) {
                        if value := currentAsArray[partIndex]; value != nil {
                            current = value
                            continue
                        }
                    }
                }
            }

            return fallback

    //  maps
        case map[string]interface{}:
            currentAsMap := current.(map[string]interface{})

            if value, ok := currentAsMap[part]; !ok {
                return fallback
            }else{
                current = value
            }
        }
    }

    return current
}



func DeepSet(data interface{}, path []string, value interface{}) interface{} {
    if len(path) == 0 {
        return data
    }

    var first = path[0]
    var rest    = make([]string, 0)

    if len(path) > 1 {
        rest = path[1:]
    }

//  Leaf Nodes
//    this is where the value we're setting actually gets set/appended
    if len(rest) == 0 {
        switch data.(type) {
    //  parent element is an ARRAY
        case []interface{}:
            return append(data.([]interface{}), value)

    //  parent element is a MAP
        case map[string]interface{}:
            dataMap := data.(map[string]interface{})
            dataMap[first] = value

            return dataMap
        }

    }else{
    //  Array Embedding
    //    this is where keys that are actually array indices get processed
    //  ================================
    //  is `first' numeric (an array index)
        if stringutil.IsInteger(rest[0]) {
            switch data.(type) {
            case map[string]interface{}:
              dataMap := data.(map[string]interface{})

          //  is the value at `first' in the map isn't present or isn't an array, create it
          //  -------->
              curVal, _ := dataMap[first]

              switch curVal.(type) {
              case []interface{}:
              default:
                  dataMap[first] = make([]interface{}, 0)
                  curVal, _ = dataMap[first]
              }
          //  <--------|


          //  recurse into our cool array and do awesome stuff with it
              dataMap[first] = DeepSet(curVal.([]interface{}), rest, value).([]interface{})
              return dataMap
            default:
              // log.Printf("WHAT %s/%s", first, rest)
            }


    //  Intermediate Map Processing
    //    this is where branch nodes get created and populated via recursion
    //    depending on the data type of the input `data', non-existent maps
    //    will be created and either set to `data[first]' (the map)
    //    or appended to `data[first]' (the array)
    //  ================================
        }else{
            switch data.(type) {
        //  handle arrays of maps
            case []interface{}:
                dataArray := data.([]interface{})

                if curIndex, err := strconv.Atoi(first); err == nil {
                    if curIndex >= len(dataArray) {
                        for add := len(dataArray); add <= curIndex; add++ {
                            dataArray = append(dataArray, make(map[string]interface{}))
                        }
                    }

                    if curIndex < len(dataArray) {
                        dataArray[curIndex] = DeepSet(dataArray[curIndex], rest, value)
                        return dataArray
                    }
                }

        //  handle good old fashioned maps-of-maps
            case map[string]interface{}:
                dataMap := data.(map[string]interface{})

            //  is the value at `first' in the map isn't present or isn't a map, create it
            //  -------->
                curVal, _ := dataMap[first]

                switch curVal.(type) {
                case map[string]interface{}:
                default:
                    dataMap[first] = make(map[string]interface{})
                    curVal, _ = dataMap[first]
                }
            //  <--------|

                dataMap[first] = DeepSet(dataMap[first], rest, value)
                return dataMap
            }
        }
    }

    return data
}
