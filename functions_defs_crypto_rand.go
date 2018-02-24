package diecast

import (
	"crypto/rand"
	"encoding/base64"

	"github.com/ghetzel/go-stockutil/stringutil"
	base58 "github.com/jbenet/go-base58"
	"github.com/spaolacci/murmur3"
)

func loadStandardFunctionsCryptoRand(rv FuncMap) {
	// fn random: Return a random array of *n* bytes. The random source used is suitable for
	//            cryptographic purposes.
	rv[`random`] = func(count int) ([]byte, error) {
		output := make([]byte, count)
		if _, err := rand.Read(output); err == nil {
			return output, nil
		} else {
			return nil, err
		}
	}

	// fn uuid: Generate a new Version 4 UUID.
	rv[`uuid`] = func() string {
		return stringutil.UUID().String()
	}

	// fn uuidRaw: Generate the raw bytes of a new Version 4 UUID.
	rv[`uuidRaw`] = func() []byte {
		return stringutil.UUID().Bytes()
	}

	// fn base32: Encode the *input* bytes with the Base32 encoding scheme.
	rv[`base32`] = func(input []byte) string {
		return Base32Alphabet.EncodeToString(input)
	}

	// fn base58: Encode the *input* bytes with the Base58 (Bitcoin alphabet) encoding scheme.
	rv[`base58`] = func(input []byte) string {
		return base58.Encode(input)
	}

	// fn base64: Encode the *input* bytes with the Base64 encoding scheme.  Optionally specify
	//            the encoding mode: one of "padded", "url", "url-padded", or empty (unpadded, default).
	rv[`base64`] = func(input []byte, encoding ...string) string {
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
	}

	// fn murmur3: hash the *input* data using the Murmur 3 algorithm.
	rv[`murmur3`] = func(input interface{}) (uint64, error) {
		if v, err := stringutil.ToString(input); err == nil {
			return murmur3.Sum64([]byte(v)), nil
		} else {
			return 0, err
		}
	}

	// TODO:
	// urlencode/urldecode
	// rv[`md5`] =
	// rv[`sha1`] =
	// rv[`sha256`] =
	// rv[`sha384`] =
	// rv[`sha512`] =
}
