package stringutil

import (
    "testing"
)

func TestToBytes(t *testing.T) {
    expected := map[string]map[string]float64{
    //  numeric passthrough (no suffix)
        ``: map[string]float64{
            `-1`:                   -1,
            `0`:                    0,
            `1`:                    1,
            `4611686018427387903`:  4611686018427387903,
            `4611686018427387904`:  4611686018427387904,
            `4611686018427387905`:  4611686018427387905,
            `9223372036854775807`:  9223372036854775807,   // beyond this overflows the positive int64 bound
            `-4611686018427387903`: -4611686018427387903,
            `-4611686018427387904`: -4611686018427387904,
            `-4611686018427387905`: -4611686018427387905,
            `-9223372036854775807`: -9223372036854775807,
            `-9223372036854775808`: -9223372036854775808,  // beyond this overflows the negative int64 bound
        },

    //  suffix: b/B
        `b`: map[string]float64{
            `-1`:                   -1,
            `0`:                    0,
            `1`:                    1,
            `4611686018427387903`:  4611686018427387903,
            `4611686018427387904`:  4611686018427387904,
            `4611686018427387905`:  4611686018427387905,
            `9223372036854775807`:  9223372036854775807,
            `-4611686018427387903`: -4611686018427387903,
            `-4611686018427387904`: -4611686018427387904,
            `-4611686018427387905`: -4611686018427387905,
            `-9223372036854775807`: -9223372036854775807,
            `-9223372036854775808`: -9223372036854775808,
        },

    //  suffix: k/K
        `k`: map[string]float64{
            `-1`:                   -1024,
            `0`:                    0,
            `1`:                    1024,
            `0.5`:                  512,
            `2`:                    2048,
            `9007199254740992`:     9223372036854775808,
        },

    //  suffix: m/M
        `m`: map[string]float64{
            `-1`:                  -1048576,
            `0`:                   0,
            `1`:                   1048576,
            `0.5`:                 524288,
            `8796093022208`:       9223372036854775808,
        },

    //  suffix: g/G
        `g`: map[string]float64{
            `-1`:                  -1073741824,
            `0`:                   0,
            `1`:                   1073741824,
            `0.5`:                 536870912,
            `8589934592`:          9223372036854775808,
        },

    //  suffix: t/T
        `t`: map[string]float64{
            `-1`:                  -1099511627776,
            `0`:                   0,
            `1`:                   1099511627776,
            `0.5`:                 549755813888,
            `8388608`:             9223372036854775808,
        },

    //  suffix: p/P
        `p`: map[string]float64{
            `-1`:                  -1125899906842624,
            `0`:                   0,
            `1`:                   1125899906842624,
            `0.5`:                 562949953421312,
            `8192`:                9223372036854775808,
        },

    //  suffix: e/E
        `e`: map[string]float64{
            `-1`:                  -1152921504606846976,
            `0`:                   0,
            `1`:                   1152921504606846976,
            `0.5`:                 576460752303423488,
            `8`:                   9223372036854775808,
        },

    //  suffix: z/Z
        `z`: map[string]float64{
            `-1`:                  -1180591620717411303424,
            `0`:                   0,
            `1`:                   1180591620717411303424,
            `0.5`:                 590295810358705651712,
        },

    //  suffix: y/Y
        `y`: map[string]float64{
            `-1`:                  -1208925819614629174706176,
            `0`:                   0,
            `1`:                   1208925819614629174706176,
            `0.5`:                 604462909807314587353088,
        },
    }

    testExpectations := func(expectedValues map[string]float64, appendToInput string){
        for in, out := range expectedValues {
            in = in + appendToInput

            if v, err := ToBytes(in); err == nil {
                if v != out {
                    t.Errorf("Conversion error on '%s': expected %f, got %f", in, out, v)
                }
            }else{
                t.Errorf("Got error converting '%s' to bytes: %v", in, err)
            }
        }
    }

    for suffix, expectations := range expected {
        testExpectations(expectations, suffix)

    //  only unleash testing hell on higher-order conversions
        if suffix != `` && suffix != `b` {
            testExpectations(expectations, suffix+`b`)
            testExpectations(expectations, suffix+`B`)
            testExpectations(expectations, suffix+`ib`)
            testExpectations(expectations, suffix+`iB`)
        }
    }

    if v, err := ToBytes(`potato`); err == nil {
        t.Errorf("Value 'potato' inexplicably returned a value (%v)", v)
    }

    if v, err := ToBytes(`potatoG`); err == nil {
        t.Errorf("Value 'potatoG' inexplicably returned a value (%v)", v)
    }

    if v, err := ToBytes(`123X`); err == nil {
        t.Errorf("Invalid SI suffix 'X' did not error, got: %v", v)
    }
}
