package diecast

import (
	"crypto/md5"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"
	"math"
	mrand "math/rand"
	"strings"

	"github.com/ghetzel/go-stockutil/stringutil"
	"github.com/ghetzel/go-stockutil/typeutil"
	"github.com/spaolacci/murmur3"
)

func hashingAlgo(alg string) (hash.Hash, error) {
	alg = strings.ToLower(alg)

	switch alg {
	case `sha1`:
		return sha1.New(), nil
	case `sha224`:
		return sha256.New224(), nil
	case `sha256`:
		return sha256.New(), nil
	case `sha384`:
		return sha512.New384(), nil
	case `sha512`:
		return sha512.New(), nil
	case `md5`:
		return md5.New(), nil
	default:
		return nil, fmt.Errorf("unknown algorithm %q", alg)
	}
}

func hashTheThing(fn string, input interface{}) (string, error) {
	if hasher, err := hashingAlgo(fn); err == nil {
		var data = typeutil.Bytes(input)
		return hex.EncodeToString(hasher.Sum(data)), nil
	} else {
		return ``, err
	}
}

func loadStandardFunctionsCryptoRand(funcs FuncMap, server *Server) funcGroup {
	return funcGroup{
		Name: `Hashing and Cryptography`,
		Description: `These functions provide basic cryptographic and non-cryptographic functions, ` +
			`including cryptographically-secure random number generation.`,
		Functions: []funcDef{
			{
				Name:    `murmur3`,
				Summary: `Hash the given data using the Murmur3 algorithm.`,
				Function: func(input interface{}) uint64 {
					return murmur3.Sum64(toBytes(input))
				},
			}, {
				Name:    `hash`,
				Summary: `Return the hash if the given value using the specified algorithm.`,
				Arguments: []funcArg{
					{
						Name:        `cleartext`,
						Type:        `string`,
						Description: `The value to perform a one-way hash operation on.`,
					}, {
						Name:        `algorithm`,
						Type:        `string`,
						Description: `The hash algorithm to use.`,
						Valid: []funcArg{
							{
								Name:        `sha1`,
								Description: `SHA-1 algorithm`,
							}, {
								Name:        `sha224`,
								Description: `SHA-224 algorithm`,
							}, {
								Name:        `sha256`,
								Description: `SHA-256 algorithm`,
							}, {
								Name:        `sha384`,
								Description: `SHA-384 algorithm`,
							}, {
								Name:        `sha512`,
								Description: `SHA-512 algorithm`,
							}, {
								Name:        `md5`,
								Description: `MD5 algorithm`,
							}, {
								Name:        `murmur3`,
								Description: `Murmur3 algorithm`,
							},
						},
					},
				},
				Function: func(input interface{}, alg string) (string, error) {
					return hashTheThing(alg, input)
				},
			}, {
				Name:    `md5`,
				Summary: `Return the MD5 hash of the given value.`,
				Arguments: []funcArg{
					{
						Name:        `cleartext`,
						Type:        `string`,
						Description: `The value to perform a one-way hash operation on.`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `md5 "p@ssw0rd!"`,
						Return: `d5ec75d5fe70d428685510fae36492d9`,
					},
				},
				Function: func(input interface{}) (string, error) {
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
				Examples: []funcExample{
					{
						Code:   `sha1 "p@ssw0rd!"`,
						Return: `ee7161e0fe1a06be63f515302806b34437563c9e`,
					},
				},
				Function: func(input interface{}) (string, error) {
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
				Examples: []funcExample{
					{
						Code:   `sha224 "p@ssw0rd!"`,
						Return: `2d2e8b944f53164ee0aa8b1f98d75713c1b1bc6b9dd67591ef0a29e0`,
					},
				},
				Function: func(input interface{}) (string, error) {
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
				Examples: []funcExample{
					{
						Code:   `sha256 "p@ssw0rd!"`,
						Return: `df2191783c6f13274b7c54330a370d0480e82a8a54069b69de73cbfa69f8ea08`,
					},
				},
				Function: func(input interface{}) (string, error) {
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
				Examples: []funcExample{
					{
						Code:   `sha384 "p@ssw0rd!"`,
						Return: `d6d02abf2b495a6e4350fd985075c88e5a6807f8f79634ddde8529507a6145cb832f40fe0220f2af242a8a4b451fb7fc`,
					},
				},
				Function: func(input interface{}) (string, error) {
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
				Examples: []funcExample{
					{
						Code:   `sha512 "p@ssw0rd!"`,
						Return: `0f8ea05dd2936700d8f23d7ceb0c7dde03e8dd2dcac714eb465c658412600457ebd143bbf8a00eed47fa0a0677cf2f2ad08f882173546a647c6802ecb19aeeb9`,
					},
				},
				Function: func(input interface{}) (string, error) {
					return hashTheThing(`sha512`, input)
				},
			}, {
				Name:    `random`,
				Summary: `Generates a random number.`,
				Arguments: []funcArg{
					{
						Name:        `lower`,
						Type:        `integer`,
						Optional:    true,
						Description: `If specified, the number generated will be greater than or equal to this number.`,
					}, {
						Name:        `upper`,
						Type:        `integer`,
						Optional:    true,
						Description: `If specified, the number generated will be strictly less than this number.`,
					},
				},
				Function: func(bounds ...interface{}) int64 {
					var min = int64(0)
					var max = int64(math.MaxInt64)

					switch len(bounds) {
					case 2:
						max = typeutil.Int(bounds[1])
						fallthrough
					case 1:
						min = typeutil.Int(bounds[0])
					}

					var delta = (max - min)

					return (mrand.Int63n(delta) + min)
				},
			}, {
				Name: `randomBytes`,
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
					var output = make([]byte, count)
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
				Name:    `hmac`,
				Summary: `Generate an HMAC signature for the given input, secret string, and (optionally) hashing algorithm.`,
				Arguments: []funcArg{
					{
						Name:        `input`,
						Type:        `string, bytes`,
						Description: `The input to generate an HMAC signature for.`,
					}, {
						Name:        `secret`,
						Type:        `string`,
						Description: `The secret string to use for generating the signature.`,
					}, {
						Name:        `algorithm`,
						Type:        `string`,
						Description: `The name of the hashing algorithm to use for signing.`,
						Default:     `sha1`,
					},
				},
				Function: func(input interface{}, secret string, alg ...string) ([]byte, error) {
					if hasher, err := hashingAlgo(typeutil.OrString(alg, `sha1`)); err == nil {
						return hasher.Sum(typeutil.Bytes(input)), nil
					} else {
						return nil, err
					}
				},
			},
		},
	}
}
