package diecast

import (
	"crypto/md5"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"fmt"

	"github.com/ghetzel/go-stockutil/stringutil"
	"github.com/ghetzel/go-stockutil/typeutil"
	base58 "github.com/jbenet/go-base58"
	"github.com/spaolacci/murmur3"
)

func hashTheThing(fn string, input interface{}) ([]byte, error) {
	data := []byte(typeutil.String(input))

	switch fn {
	case `md5`:
		out := md5.Sum(data)
		return out[:], nil
	case `sha1`:
		out := sha1.Sum(data)
		return out[:], nil
	case `sha224`:
		out := sha256.Sum224(data)
		return out[:], nil
	case `sha256`:
		out := sha256.Sum256(data)
		return out[:], nil
	case `sha384`:
		out := sha512.Sum384(data)
		return out[:], nil
	case `sha512`:
		out := sha512.Sum512(data)
		return out[:], nil
	default:
		return nil, fmt.Errorf("Unimplemented hashing function %q", fn)
	}
}

func loadStandardFunctionsCryptoRand(funcs FuncMap) funcGroup {
	// TODO:
	// urlencode/urldecode

	return funcGroup{
		Name: `Hashing and Cryptography`,
		Description: `These functions provide basic cryptographic and non-cryptographic functions, ` +
			`including cryptographically-secure random number generation.`,
		Functions: []funcDef{
			{
				Name:    `md5`,
				Summary: `Return the MD5 hash of the given value.`,
				Arguments: []funcArg{
					{
						Name:        `cleartext`,
						Type:        `string`,
						Description: `The value to perform a one-way hash operation on.`,
					},
				},
				Function: func(input interface{}) ([]byte, error) {
					return hashTheThing(`md5`, input)
				},
			}, {
				Name:    `sha1`,
				Summary: `Return the SHA-1 hash of the given value.`,
				Arguments: []funcArg{
					{
						Name:        `cleartext`,
						Type:        `string`,
						Description: `The value to perform a one-way hash operation on.`,
					},
				},
				Function: func(input interface{}) ([]byte, error) {
					return hashTheThing(`sha1`, input)
				},
			}, {
				Name:    `sha224`,
				Summary: `Return the SHA-224 hash of the given value.`,
				Arguments: []funcArg{
					{
						Name:        `cleartext`,
						Type:        `string`,
						Description: `The value to perform a one-way hash operation on.`,
					},
				},
				Function: func(input interface{}) ([]byte, error) {
					return hashTheThing(`sha224`, input)
				},
			}, {
				Name:    `sha256`,
				Summary: `Return the SHA-256 hash of the given value.`,
				Arguments: []funcArg{
					{
						Name:        `cleartext`,
						Type:        `string`,
						Description: `The value to perform a one-way hash operation on.`,
					},
				},
				Function: func(input interface{}) ([]byte, error) {
					return hashTheThing(`sha256`, input)
				},
			}, {
				Name:    `sha384`,
				Summary: `Return the SHA-384 hash of the given value.`,
				Arguments: []funcArg{
					{
						Name:        `cleartext`,
						Type:        `string`,
						Description: `The value to perform a one-way hash operation on.`,
					},
				},
				Function: func(input interface{}) ([]byte, error) {
					return hashTheThing(`sha384`, input)
				},
			}, {
				Name:    `sha512`,
				Summary: `Return the SHA-512 hash of the given value.`,
				Arguments: []funcArg{
					{
						Name:        `cleartext`,
						Type:        `string`,
						Description: `The value to perform a one-way hash operation on.`,
					},
				},
				Function: func(input interface{}) ([]byte, error) {
					return hashTheThing(`sha512`, input)
				},
			}, {
				Name: `random`,
				Summary: `Return a random array of _n_ bytes. The random source used is ` +
					`suitable for cryptographic purposes.`,
				Arguments: []funcArg{
					{
						Name:        `count`,
						Type:        `integer`,
						Description: `The size of the output array of random bytes to return.`,
					},
				},
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
				Name:    `hex`,
				Summary: `Encode the given value as a hexadecimal string.`,
				Arguments: []funcArg{
					{
						Name: `input`,
						Type: `string, bytes`,
						Description: `The value to encode. If a byte array is provided, it will be encoded in ` +
							`hexadecimal. If a string is provided, it will converted to a byte array first, then encoded.`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `hex "hello"`,
						Return: `68656c6c6f`,
					},
				},
				Function: func(input interface{}) (string, error) {
					return hex.EncodeToString(toBytes(input)), nil
				},
			}, {
				Name:    `base32`,
				Summary: `Encode the given bytes with the Base32 encoding scheme.`,
				Arguments: []funcArg{
					{
						Name: `input`,
						Type: `string, bytes`,
						Description: `The value to encode. If a byte array is provided, it will be encoded directly. ` +
							`If a string is provided, it will converted to a byte array first, then encoded.`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `base32 "hello"`,
						Return: `nbswy3dp`,
					},
				},
				Function: func(input interface{}) string {
					return Base32Alphabet.EncodeToString(toBytes(input))
				},
			}, {
				Name:    `base58`,
				Summary: `Encode the given bytes with the Base58 (Bitcoin alphabet) encoding scheme.`,
				Function: func(input interface{}) string {
					return base58.Encode(toBytes(input))
				},
			}, {
				Name:    `base64`,
				Summary: `Encode the given bytes with the Base64 encoding scheme.  Optionally specify the encoding mode: one of "padded", "url", "url-padded", or empty (unpadded, default).`,
				Arguments: []funcArg{
					{
						Name: `input`,
						Type: `string, bytes`,
						Description: `The value to encode. If a byte array is provided, it will be encoded directly. ` +
							`If a string is provided, it will converted to a byte array first, then encoded.`,
					}, {
						Name:        `encoding`,
						Type:        `string`,
						Optional:    true,
						Description: `Specify an encoding option for generating the Base64 representation.`,
						Valid: []funcArg{
							{
								Name:        `standard`,
								Description: `Standard Base-64 encoding scheme, no padding.`,
							}, {
								Name:        `padded`,
								Description: `Standard Base-64 encoding scheme, preserves padding.`,
							}, {
								Name:        `url`,
								Description: `Encoding that can be used in URLs and filenames, no padding.`,
							}, {
								Name:        `url-padded`,
								Description: `Encoding that can be used in URLs and filenames, preserves padding.`,
							},
						},
					},
				},
				Examples: []funcExample{
					{
						Code:   `base64 "hello?yes=this&is=dog#"`,
						Return: `aGVsbG8/eWVzPXRoaXMmaXM9ZG9nIw`,
					}, {
						Description: `This is identical to the above example, but with the encoding explicitly specified.`,
						Code:        `base64 "hello?yes=this&is=dog#" "standard"`,
						Return:      `aGVsbG8/eWVzPXRoaXMmaXM9ZG9nIw`,
					}, {
						Code:   `base64 "hello?yes=this&is=dog#" "padded"`,
						Return: `aGVsbG8/eWVzPXRoaXMmaXM9ZG9nIw==`,
					}, {
						Code:   `base64 "hello?yes=this&is=dog#" "url"`,
						Return: `aGVsbG8_eWVzPXRoaXMmaXM9ZG9nIw`,
					}, {
						Code:   `base64 "hello?yes=this&is=dog#" "url-padded"`,
						Return: `aGVsbG8_eWVzPXRoaXMmaXM9ZG9nIw==`,
					},
				},
				Function: func(input interface{}, encoding ...string) string {
					if len(encoding) == 0 {
						encoding = []string{`standard`}
					}

					switch encoding[0] {
					case `padded`:
						return base64.StdEncoding.EncodeToString(toBytes(input))
					case `url`:
						return base64.RawURLEncoding.EncodeToString(toBytes(input))
					case `url-padded`:
						return base64.URLEncoding.EncodeToString(toBytes(input))
					default:
						return base64.RawStdEncoding.EncodeToString(toBytes(input))
					}
				},
			}, {
				Name:    `murmur3`,
				Summary: `Hash the given data using the Murmur 3 algorithm.`,
				Function: func(input interface{}) (uint64, error) {
					return murmur3.Sum64(toBytes(input)), nil
				},
			},
		},
	}
}
