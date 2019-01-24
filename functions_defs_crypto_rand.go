package diecast

import (
	"crypto/rand"
	"encoding/base64"

	"github.com/ghetzel/go-stockutil/stringutil"
	base58 "github.com/jbenet/go-base58"
	"github.com/spaolacci/murmur3"
)

func loadStandardFunctionsCryptoRand() funcGroup {
	return funcGroup{
		Name: `Hashing and Cryptography`,
		Description: `These functions provide basic cryptographic and non-cryptographic functions, ` +
			`including cryptographically-secure random number generation.`,
		Functions: []funcDef{
			{
				{
					Name: `random`,
					Summary: `Return a random array of _n_ bytes. The random source used is ` +
						`suitable for cryptographic purposes.`,
					Function: func(count int) ([]byte, error) {
						output := make([]byte, count)
						if _, err := rand.Read(output); err == nil {
							return output, nil
						} else {
							return nil, err
						}
					},
				}, {
					Name:    `uuid`,
					Summary: `Generate a new Version 4 UUID as a string.`,
					Function: func() string {
						return stringutil.UUID().String()
					},
				}, {
					Name:    `uuidRaw`,
					Summary: `Generate the raw bytes of a new Version 4 UUID value.`,
					Function: func() []byte {
						return stringutil.UUID().Bytes()
					},
				}, {
					Name:    `base32`,
					Summary: `Encode the *input* bytes with the Base32 encoding scheme.`,
					Function: func(input []byte) string {
						return Base32Alphabet.EncodeToString(input)
					},
				}, {
					Name:    `base58`,
					Summary: `Encode the *input* bytes with the Base58 (Bitcoin alphabet) encoding scheme.`,
					Function: func(input []byte) string {
						return base58.Encode(input)
					},
				}, {
					Name:    `base64`,
					Summary: `Encode the *input* bytes with the Base64 encoding scheme.  Optionally specify the encoding mode: one of "padded", "url", "url-padded", or empty (unpadded, default).`,
					Function: func(input []byte, encoding ...string) string {
						if len(encoding) == 0 {
							encoding = []string{`standard`}
						}

						switch encoding[0] {
						case `padded`:
							return base64.StdEncoding.EncodeToString(input)
						case `url`:
							return base64.RawURLEncoding.EncodeToString(input)
						case `url-padded`:
							return base64.URLEncoding.EncodeToString(input)
						default:
							return base64.RawStdEncoding.EncodeToString(input)
						}
					},
				}, {
					Name:    `murmur3`,
					Summary: `Hash the given data using the Murmur 3 algorithm.`,
					Function: func(input interface{}) (uint64, error) {
						if v, err := stringutil.ToString(input); err == nil {
							return murmur3.Sum64([]byte(v)), nil
						} else {
							return 0, err
						}
					},
				},
			},
		},
	}
}
