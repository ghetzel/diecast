package sliceutil

import (
    "testing"
)

func TestContainsString(t *testing.T) {
    input := []string{ "one", "three", "five" }


    if ContainsString(input, "one") != true {
        t.Errorf("Input slice should contain 'one'")
    }

    if ContainsString(input, "two") == true {
        t.Errorf("Input slice should not contain 'two'")
    }

    if ContainsString([]string{}, "one") == true {
        t.Errorf("Empty slice should not contain 'one'")
    }

    if ContainsString([]string{}, "two") == true {
        t.Errorf("Input slice should not contain 'two'")
    }

    if ContainsString([]string{}, "") == true {
        t.Errorf("Input slice should not contain ''")
    }
}