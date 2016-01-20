package maputil

import (
    "testing"
    "fmt"
)

func TestDiffuseOneTierScalar(t *testing.T) {
    var err error

    input  := make(map[string]interface{})
    output := make(map[string]interface{})

    input["id"] = "test"
    input["enabled"] = true
    input["float"] = 2.7

    if output, err = DiffuseMap(input, "."); err != nil {
        t.Errorf("%s\n", err)
    }

    if v, ok := output["id"]; !ok || v != "test" {
        t.Errorf("Incorrect value '%s' for key %s", v, "id")
    }

    if v, ok := output["enabled"]; !ok || v != true {
        t.Errorf("Incorrect value '%s' for key %s", v, "enabled")
    }

    if v, ok := output["float"]; !ok || v != 2.7 {
        t.Errorf("Incorrect value '%s' for key %s", v, "float")
    }
}


func TestDiffuseOneTierComplex(t *testing.T) {
    var err error

    input  := make(map[string]interface{})
    output := make(map[string]interface{})


    input["array"] = []string{ "first", "third", "fifth" }
    input["numary"] = []int{ 9, 7, 3 }
    input["things"] = map[string]int{ "one": 1, "two": 2, "three": 3 }

    if output, err = DiffuseMap(input, "."); err != nil {
        t.Errorf("%s\n", err)
    }


//  test string array
    if _, ok := output["array"]; !ok  {
        t.Errorf("Key %s is missing from output", "array")
    }

    for i, v := range output["array"].([]string) {
        if v != input["array"].([]string)[i] {
            t.Errorf("Incorrect value '%s' for key %s[%d]", v, "array", i)
        }
    }

//  test int array
    if _, ok := output["numary"]; !ok  {
        t.Errorf("Key %s is missing from output", "numary")
    }

    for i, v := range output["numary"].([]int) {
        if v != input["numary"].([]int)[i] {
            t.Errorf("Incorrect value '%s' for key %s[%d]", v, "numary", i)
        }
    }

//  test string-int map
    if _, ok := output["things"]; !ok  {
        t.Errorf("Key %s is missing from output", "things")
    }

    for k, v := range output["things"].(map[string]int) {
        if inputValue, ok := input["things"].(map[string]int)[k]; !ok || v != inputValue {
            t.Errorf("Incorrect value '%s' for key %s[%s]", v, "things", k)
        }
    }
}



func TestDiffuseMultiTierScalar(t *testing.T) {
    var err error

    input  := make(map[string]interface{})
    output := make(map[string]interface{})

    input["items.0"] = 54
    input["items.1"] = 77
    input["items.2"] = 82

    if output, err = DiffuseMap(input, "."); err != nil {
        t.Errorf("%s\n", err)
    }

    if i_items, ok := output["items"]; ok {
        items := i_items.([]interface{})

        for i, v := range []int{ 54, 77, 82 } {
            if len(items) <= i || items[i].(int) != v {
                t.Errorf("Output items[%d] != %v", i, v)
            }
        }
    }else{
        t.Errorf("Key 'items' is missing from output: %v", output)
    }
}


func TestDiffuseMultiTierComplex(t *testing.T) {
    var err error

    input  := make(map[string]interface{})
    output := make(map[string]interface{})

    input["items.0.name"] = "First"
    input["items.0.age"]  = 54
    input["items.1.name"] = "Second"
    input["items.1.age"]  = 77
    input["items.2.name"] = "Third"
    input["items.2.age"]  = 82

    if output, err = DiffuseMap(input, "."); err != nil {
        t.Errorf("%s\n", err)
    }

    if i_items, ok := output["items"]; ok {
        items := i_items.([]interface{})

        if len(items) != 3 {
            t.Errorf("Key 'items' should be an array with 3 elements, got %v", i_items)
        }

        for item_id, obj := range items {
            for k, v := range obj.(map[string]interface{}) {
                if inValue, ok := input[fmt.Sprintf("items.%d.%s", item_id, k)]; !ok || inValue != v {
                    t.Errorf("Key %s Incorrect, expected %s, got %s", fmt.Sprintf("items.%d.%s", item_id, k), inValue, v)
                }
            }
        }
    }else{
        t.Errorf("Key 'items' is missing from output: %v", output)
    }
}


func TestDiffuseMultiTierMixed(t *testing.T) {
    var err error

    input  := make(map[string]interface{})
    output := make(map[string]interface{})

    input["items.0.tags"] = []string{ "base", "other" }
    input["items.1.tags"] = []string{ "thing", "still-other", "more-other" }
    input["items.2.tags"] = []string{ "last" }

    if output, err = DiffuseMap(input, "."); err != nil {
        t.Errorf("%s\n", err)
    }

    if i_items, ok := output["items"]; ok {
        items := i_items.([]interface{})

        if len(items) != 3 {
            t.Errorf("Key 'items' should be an array with 3 elements, got %v", i_items)
        }

        for item_id, obj := range items {
            for k, v := range obj.(map[string]interface{}) {
                vAry := v.([]string)

                if inValue, ok := input[fmt.Sprintf("items.%d.%s", item_id, k)]; !ok {
                    t.Errorf("Key %s Incorrect, expected %s, got %s", fmt.Sprintf("items.%d.%s", item_id, k), inValue, v)
                }else{
                    inValueAry := inValue.([]string)

                    for i, vAryV := range vAry {

                        if vAryV != inValueAry[i] {
                            t.Errorf("Key %s[%d] Incorrect, expected %s, got %s", fmt.Sprintf("items.%d.%s", item_id, k), i, inValueAry[i], vAryV)
                        }
                    }
                }
            }
        }
    }else{
        t.Errorf("Key 'items' is missing from output: %v", output)
    }
}