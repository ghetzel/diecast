package maputil

import (
    "testing"
    // "fmt"
)

func TestDiffuseTypedOneTierScalar(t *testing.T) {
    var errs []error

    input  := make(map[string]interface{})
    output := make(map[string]interface{})

    input["str:id"] = "test"
    input["name"] = "default-string"
    input["bool:enabled"] = "true"
    input["float:float"] = "2.7"

    if output, errs = DiffuseMapTyped(input, ".", ":"); len(errs) > 0 {
        for _, err := range errs {
            t.Errorf("%s\n", err)
        }
    }

    if v, ok := output["id"]; !ok || v != "test" {
        t.Errorf("Incorrect value '%s' for key %s", v, "id")
    }

    if v, ok := output["name"]; !ok || v != "default-string" {
        t.Errorf("Incorrect value '%s' for key %s", v, "default-string")
    }

    if v, ok := output["enabled"]; !ok || v != true {
        t.Errorf("Incorrect value '%s' for key %s", v, "enabled")
    }

    if v, ok := output["float"]; !ok || v != 2.7 {
        t.Errorf("Incorrect value '%s' for key %s", v, "float")
    }
}


func TestDiffuseTypedOneTierComplex(t *testing.T) {
    var errs []error

    input  := make(map[string]interface{})
    output := make(map[string]interface{})


    input["str:array"] = []string{ "first", "third", "fifth" }
    input["array2"] = []string{ "first", "third", "fifth" }
    input["int:numary.0"] = "9"
    input["int:numary.1"] = "7"
    input["int:numary.2"] = "3"
    input["int:things.one"] = "1"
    input["int:things.two"] = "2"
    input["int:things.three"] = "3"

    if output, errs = DiffuseMapTyped(input, ".", ":"); len(errs) > 0 {
        for _, err := range errs {
            t.Errorf("%s\n", err)
        }
    }

//  test string array
    if _, ok := output["array"]; !ok  {
        t.Errorf("Key %s is missing from output", "array")
        return
    }

    for i, v := range output["array"].([]string) {
        if v != input["str:array"].([]string)[i] {
            t.Errorf("Incorrect value '%s' for key %s[%d]", v, "array", i)
        }
    }

    if _, ok := output["array"]; !ok  {
        t.Errorf("Key %s is missing from output", "array")
    }

    for i, v := range output["array2"].([]string) {
        if v != input["array2"].([]string)[i] {
            t.Errorf("Incorrect value '%s' for key %s[%d]", v, "array", i)
        }
    }

//  test int array
    if _, ok := output["numary"]; !ok  {
        t.Errorf("Key %s is missing from output", "numary")
    }

    if l := len(output["numary"].([]interface{})); l != 3 {
        t.Errorf("Incorrect length for numary; expected 3, got %d", l)
    }

    if a := output["numary"].([]interface{}); a[0].(int64) != 9 {
        t.Errorf("Expected numary[0] = 9, got %v", a[0])
    }

    if a := output["numary"].([]interface{}); a[1].(int64) != 7 {
        t.Errorf("Expected numary[1] = 7, got %v", a[1])
    }

    if a := output["numary"].([]interface{}); a[2].(int64) != 3 {
        t.Errorf("Expected numary[2] = 3, got %v", a[2])
    }

//  test string-int map
    if _, ok := output["things"]; !ok  {
        t.Errorf("Key %s is missing from output", "things")
    }

    for k, v := range output["things"].(map[string]interface{}) {
        switch k {
        case `one`:
            if v.(int64) != 1 {
                t.Errorf("Expected things['one'] = 1, got %v", v)
            }
        case `two`:
            if v.(int64) != 2 {
                t.Errorf("Expected things['two'] = 2, got %v", v)
            }
        case `three`:
            if v.(int64) != 3 {
                t.Errorf("Expected things['three'] = 3, got %v", v)
            }
        }
    }
}



// func TestDiffuseMultiTierScalar(t *testing.T) {
//     var err error

//     input  := make(map[string]interface{})
//     output := make(map[string]interface{})

//     input["items.0"] = 54
//     input["items.1"] = 77
//     input["items.2"] = 82

//     if output, err = DiffuseMap(input, "."); err != nil {
//         t.Errorf("%s\n", err)
//     }

//     if i_items, ok := output["items"]; ok {
//         items := i_items.([]interface{})

//         for i, v := range []int{ 54, 77, 82 } {
//             if len(items) <= i || items[i].(int) != v {
//                 t.Errorf("Output items[%d] != %v", i, v)
//             }
//         }
//     }else{
//         t.Errorf("Key 'items' is missing from output: %v", output)
//     }
// }


// func TestDiffuseMultiTierComplex(t *testing.T) {
//     var err error

//     input  := make(map[string]interface{})
//     output := make(map[string]interface{})

//     input["items.0.name"] = "First"
//     input["items.0.age"]  = 54
//     input["items.1.name"] = "Second"
//     input["items.1.age"]  = 77
//     input["items.2.name"] = "Third"
//     input["items.2.age"]  = 82

//     if output, err = DiffuseMap(input, "."); err != nil {
//         t.Errorf("%s\n", err)
//     }

//     if i_items, ok := output["items"]; ok {
//         items := i_items.([]interface{})

//         if len(items) != 3 {
//             t.Errorf("Key 'items' should be an array with 3 elements, got %v", i_items)
//         }

//         for item_id, obj := range items {
//             for k, v := range obj.(map[string]interface{}) {
//                 if inValue, ok := input[fmt.Sprintf("items.%d.%s", item_id, k)]; !ok || inValue != v {
//                     t.Errorf("Key %s Incorrect, expected %s, got %s", fmt.Sprintf("items.%d.%s", item_id, k), inValue, v)
//                 }
//             }
//         }
//     }else{
//         t.Errorf("Key 'items' is missing from output: %v", output)
//     }
// }


// func TestDiffuseMultiTierMixed(t *testing.T) {
//     var err error

//     input  := make(map[string]interface{})
//     output := make(map[string]interface{})

//     input["items.0.tags"] = []string{ "base", "other" }
//     input["items.1.tags"] = []string{ "thing", "still-other", "more-other" }
//     input["items.2.tags"] = []string{ "last" }

//     if output, err = DiffuseMap(input, "."); err != nil {
//         t.Errorf("%s\n", err)
//     }

//     if i_items, ok := output["items"]; ok {
//         items := i_items.([]interface{})

//         if len(items) != 3 {
//             t.Errorf("Key 'items' should be an array with 3 elements, got %v", i_items)
//         }

//         for item_id, obj := range items {
//             for k, v := range obj.(map[string]interface{}) {
//                 vAry := v.([]string)

//                 if inValue, ok := input[fmt.Sprintf("items.%d.%s", item_id, k)]; !ok {
//                     t.Errorf("Key %s Incorrect, expected %s, got %s", fmt.Sprintf("items.%d.%s", item_id, k), inValue, v)
//                 }else{
//                     inValueAry := inValue.([]string)

//                     for i, vAryV := range vAry {

//                         if vAryV != inValueAry[i] {
//                             t.Errorf("Key %s[%d] Incorrect, expected %s, got %s", fmt.Sprintf("items.%d.%s", item_id, k), i, inValueAry[i], vAryV)
//                         }
//                     }
//                 }
//             }
//         }
//     }else{
//         t.Errorf("Key 'items' is missing from output: %v", output)
//     }
// }