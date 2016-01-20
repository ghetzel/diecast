package maputil

import (
    "testing"
)

func TestCoalesceTypedOneTierScalar(t *testing.T) {
    var errs []error

    input  := make(map[string]interface{})
    output := make(map[string]interface{})

    input["id"] = "test"
    input["enabled"] = true
    input["float"] = 2.7

    if output, errs = CoalesceMapTyped(input, ".", ":"); len(errs) > 0 {
        for _, err := range errs {
            t.Errorf("%s\n", err)
        }
    }

    if v, ok := output["str:id"]; !ok || v != "test" {
        t.Errorf("Incorrect value '%s' for key %s", v, "str:id")
    }

    if v, ok := output["bool:enabled"]; !ok || v != "true" {
        t.Errorf("Incorrect value '%s' for key %s", v, "bool:enabled")
    }

    if v, ok := output["float:float"]; !ok || v != "2.7" {
        t.Errorf("Incorrect value '%s' for key %s", v, "float:float")
    }
}


func TestCoalesceTypedMultiTierScalar(t *testing.T) {
    var errs []error

    input  := make(map[string]interface{})
    output := make(map[string]interface{})

    input["id"] = "top"
    input["nested"] = make(map[string]interface{})
    input["nested"].(map[string]interface{})["data"] = true
    input["nested"].(map[string]interface{})["value"] = 4.9
    input["nested"].(map[string]interface{})["awesome"] = "very yes"

    if output, errs = CoalesceMapTyped(input, ".", ":"); len(errs) > 0 {
        for _, err := range errs {
            t.Errorf("%s\n", err)
        }
    }

    if v, ok := output["str:id"]; !ok || v != "top" {
        t.Errorf("Incorrect value '%s' for key %s", v, "str:id")
    }

    if v, ok := output["bool:nested.data"]; !ok || v != "true" {
        t.Errorf("Incorrect value '%s' for key %s", v, "bool:nested.data")
    }

    if v, ok := output["float:nested.value"]; !ok || v != "4.9" {
        t.Errorf("Incorrect value '%s' for key %s", v, "float:nested.value")
    }

    if v, ok := output["str:nested.awesome"]; !ok || v != "very yes" {
        t.Errorf("Incorrect value '%s' for key %s", v, "str:nested.awesome")
    }
}


func TestCoalesceTypedTopLevelArray(t *testing.T) {
    var errs []error

    input  := make(map[string]interface{})
    output := make(map[string]interface{})

    numbers := make([]interface{}, 0)
    numbers = append(numbers, 1)
    numbers = append(numbers, 2)
    numbers = append(numbers, 3)

    input["numbers"] = numbers

    if output, errs = CoalesceMapTyped(input, ".", ":"); len(errs) > 0 {
        for _, err := range errs {
            t.Errorf("%s\n", err)
        }
    }

    if v, ok := output["int:numbers.0"]; !ok || v != "1" {
        t.Errorf("Incorrect value '%s' for key %s", v, "int:numbers.0")
    }

    if v, ok := output["int:numbers.1"]; !ok || v != "2" {
        t.Errorf("Incorrect value '%s' for key %s", v, "int:numbers.1")
    }

    if v, ok := output["int:numbers.2"]; !ok || v != "3" {
        t.Errorf("Incorrect value '%s' for key %s", v, "int:numbers.2")
    }
}


func TestCoalesceTypedArrayWithNestedMap(t *testing.T) {
    var errs []error

    input  := make(map[string]interface{})
    output := make(map[string]interface{})

    numbers := make([]interface{}, 0)
    numbers = append(numbers, map[string]interface{}{
        "name":  "test",
        "count": 2,
    })

    numbers = append(numbers, map[string]interface{}{
        "name":  "test2",
        "count": 4,
    })

    numbers = append(numbers, map[string]interface{}{
        "name":  "test3",
        "count": 8,
    })

    input["numbers"] = numbers

    if output, errs = CoalesceMapTyped(input, ".", ":"); len(errs) > 0 {
        for _, err := range errs {
            t.Errorf("%s\n", err)
        }
    }

    if v, ok := output["str:numbers.0.name"]; !ok || v != "test" {
        t.Errorf("Incorrect value '%s' for key %s", v, "numbers.0.name")
    }

    if v, ok := output["int:numbers.0.count"]; !ok || v != "2" {
        t.Errorf("Incorrect value '%s' for key %s", v, "int:numbers.0.count")
    }


    if v, ok := output["str:numbers.1.name"]; !ok || v != "test2" {
        t.Errorf("Incorrect value '%s' for key %s", v, "str:numbers.1.name")
    }

    if v, ok := output["int:numbers.1.count"]; !ok || v != "4" {
        t.Errorf("Incorrect value '%s' for key %s", v, "int:numbers.1.count")
    }


    if v, ok := output["str:numbers.2.name"]; !ok || v != "test3" {
        t.Errorf("Incorrect value '%s' for key %s", v, "str:numbers.2.name")
    }

    if v, ok := output["int:numbers.2.count"]; !ok || v != "8" {
        t.Errorf("Incorrect value '%s' for key %s", v, "int:numbers.2.count")
    }
}

