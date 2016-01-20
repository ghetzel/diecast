package stringutil

import (
    "testing"
)

func TestToString(t *testing.T) {
    testvalues := map[interface{}]string {
        int(0):  `0`,
        int(1):  `1`,
        int8(0): `0`,
        int8(1): `1`,
        int16(0): `0`,
        int16(1): `1`,
        int32(0): `0`,
        int32(1): `1`,
        int64(0): `0`,
        int64(1): `1`,

        uint(0):  `0`,
        uint(1):  `1`,
        uint8(0): `0`,
        uint8(1): `1`,
        uint16(0): `0`,
        uint16(1): `1`,
        uint32(0): `0`,
        uint32(1): `1`,
        uint64(0): `0`,
        uint64(1): `1`,

        float32(0.0): `0`,
        float32(1.0): `1`,
        float64(0.0): `0`,
        float64(1.0): `1`,
        float32(0.5): `0.5`,
        float32(1.7): `1.7`,
        float64(0.6): `0.6`,
        float64(1.2): `1.2`,
    }

    for in, out := range testvalues {
        if v, err := ToString(in); err != nil || v != out {
            t.Errorf("Value %v (%T) ToString failed: expected '%s', got '%s' (err: %v)", in, in, out, v, err)
        }
    }
}
