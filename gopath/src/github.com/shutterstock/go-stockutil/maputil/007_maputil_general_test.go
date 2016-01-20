package maputil

import (
    "testing"
    "strings"
)

func TestMapJoin(t *testing.T) {
    input := map[string]interface{}{
        `key1`: `value1`,
        `key2`: true,
        `key3`: 3,
    }

    output := Join(input, `=`, `&`)

    if output == `` {
        t.Error("Output should not be empty")
    }

    if !strings.Contains(output, `key1=value1`) {
        t.Errorf("Output should contain '%s'", `key1=value1`)
    }

    if !strings.Contains(output, `key2=true`) {
        t.Errorf("Output should contain '%s'", `key2=true`)
    }

    if !strings.Contains(output, `key3=3`) {
        t.Errorf("Output should contain '%s'", `key3=3`)
    }
}


func TestMapSplit(t *testing.T) {
    input := `key1=value1&key2=true&key3=3`

    output := Split(input, `=`, `&`)

    if len(output) == 0 {
        t.Error("Output should not be empty")
    }

    if v, ok := output[`key1`]; !ok || v != `value1` {
        t.Errorf("Output should contain key %s => '%s'", `key1` , `value1`)
    }

    if v, ok := output[`key2`]; !ok || v != `true` {
        t.Errorf("Output should contain key %s => '%s'", `key2` , `true`)
    }

    if v, ok := output[`key3`]; !ok || v != `3` {
        t.Errorf("Output should contain key %s => '%s'", `key3` , `3`)
    }
}

