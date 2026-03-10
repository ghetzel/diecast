package diecast

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-crypt/crypt"
	"github.com/go-crypt/crypt/algorithm"
	"github.com/go-crypt/crypt/algorithm/argon2"
	"github.com/go-crypt/crypt/algorithm/bcrypt"
	"github.com/go-crypt/crypt/algorithm/md5crypt"
	"github.com/go-crypt/crypt/algorithm/pbkdf2"
	"github.com/go-crypt/crypt/algorithm/plaintext"
	"github.com/go-crypt/crypt/algorithm/scrypt"
	"github.com/go-crypt/crypt/algorithm/sha1crypt"
	"github.com/go-crypt/crypt/algorithm/shacrypt"
)

var DefaultCredentialProvider = `default`
var DefaultStaticHashAlgo = `scrypt`

type Credential interface {
	BasicAuthenticate(user string, password string) error
}

type StaticCredentialProvider struct {
	staticUsers map[string]string
}

func (local *StaticCredentialProvider) HashPassword(algo string, cleartextPassword string) (string, error) {
	var hash algorithm.Hash
	var hasherr error

	switch strings.ToLower(algo) {
	case `argon2`:
		hash, hasherr = argon2.New()
	case `bcrypt`:
		hash, hasherr = bcrypt.New()
	case `md5crypt`:
		hash, hasherr = md5crypt.New()
	case `pbkdf2`:
		hash, hasherr = pbkdf2.New()
	case `plaintext`:
		hash, hasherr = plaintext.New()
	case `scrypt`:
		hash, hasherr = scrypt.New()
	case `sha1crypt`:
		hash, hasherr = sha1crypt.New()
	case `shacrypt `:
		hash, hasherr = shacrypt.New()
	default:
		return ``, fmt.Errorf("undefined algorithm %q", algo)
	}

	if hasherr == nil {
		if digest, err := hash.Hash(cleartextPassword); err == nil {
			return digest.Encode(), nil
		} else {
			return ``, err
		}
	} else {
		return ``, hasherr
	}
}

func (local *StaticCredentialProvider) AddUserCleartext(username string, cleartextPassword string) error {
	return local.AddUserWithAlgorithm(DefaultStaticHashAlgo, username, cleartextPassword)
}

func (local *StaticCredentialProvider) AddUserWithAlgorithm(algo string, username string, cleartextPassword string) error {
	if encodedDigest, err := local.HashPassword(algo, cleartextPassword); err == nil {
		return local.AddUser(username, encodedDigest)
	} else {
		return err
	}
}

func (local *StaticCredentialProvider) AddUser(username string, passwordDigest string) error {
	if len(local.staticUsers) == 0 {
		local.staticUsers = make(map[string]string)
	}

	local.staticUsers[username] = passwordDigest
	return nil
}

func (local *StaticCredentialProvider) BasicAuthenticate(user string, password string) error {
	if encodedDigest, ok := local.staticUsers[user]; ok {
		if valid, err := crypt.CheckPassword(password, encodedDigest); err == nil {
			if valid {
				return nil
			} else {
				return errors.New("invalid password")
			}
		} else {
			return err
		}
	} else {
		return errors.New("no such user")
	}
}
