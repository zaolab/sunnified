package sec

import (
	"bytes"
	"code.google.com/p/go.crypto/scrypt"
	"encoding/base64"
	"errors"
)

const DEFAULT_SALTLEN = 24

// cryptiter is used to track the crypto version since the start of sunnified.sec.auth
// when crypto used is changed, the cryptiter will increment
const cryptiter uint8 = 1
const scrypt_keylen = 64

var ErrSaltGenFailed = errors.New("Unable to generate a random salt")

type AuthPassword struct {
	config AuthPasswordConfig
}

type AuthPasswordConfig struct {
	SunnyConfig bool  `config.namespace:"sunnified.sec.auth"`
	Saltlen     uint8 `config.default:"24"`
	Strength    int8  `config.default:"5"` // strength ranges from 1-10
}

func NewAuthPassword(settings AuthPasswordConfig) *AuthPassword {
	if settings.Saltlen == 0 {
		settings.Saltlen = DEFAULT_SALTLEN
	}

	return &AuthPassword{config: settings}
}

func (this *AuthPassword) CryptPassword(pwd string) (string, error) {
	nshift, r, p := getScryptCost(this.config.Strength)
	// since n must be 2^
	n := int(1 << nshift)

	salt := GenRandomBytes(int(this.config.Saltlen))
	if salt == nil {
		return "", ErrSaltGenFailed
	}

	key, err := scrypt.Key([]byte(pwd), salt, n, r, p, scrypt_keylen)

	if err != nil {
		return "", err
	}

	bslice := make([]byte, 1, this.config.Saltlen+scrypt_keylen+3)
	bslice[0] = byte(this.config.Saltlen)
	bslice = append(bslice, salt...)
	bslice = append(bslice, byte(cryptiter), byte(this.config.Strength))
	bslice = append(bslice, key...)
	return base64.StdEncoding.EncodeToString(bslice), nil
}

func (this *AuthPassword) VerifyPassword(hashstr, pwd string) bool {
	return VerifyPassword(hashstr, pwd)
}

// when crypto is updated, the hash done using previous crypto can still be verified
// and the new hash with the updated crypto will be returned
// this allows rolling updates of new crypto hash function for password hashing
func (this *AuthPassword) VerifyPasswordAndUpdateHash(hashstr, pwd string) (bool, string) {
	ok, iter, strength, saltlen := VerifyPasswordGetMeta(hashstr, pwd)

	if ok && (iter != cryptiter || strength != this.config.Strength || saltlen != this.config.Saltlen) {
		hash, err := this.CryptPassword(pwd)
		if err == nil {
			return ok, hash
		}
	}

	return ok, ""
}

func (this *AuthPassword) HashIsOutdated(hashstr string) bool {
	key, salt, iter, strength := getKeySaltIterStrength(hashstr)
	return key == nil || iter != cryptiter || strength != this.config.Strength || len(salt) != int(this.config.Saltlen)
}

func getKeySaltIterStrength(hashstr string) ([]byte, []byte, uint8, int8) {
	hash, err := base64.StdEncoding.DecodeString(hashstr)

	if err != nil {
		return nil, nil, 0, 0
	}

	var (
		hashlen       = len(hash)
		saltlen, iter uint8
		strength      int8
		salt          []byte
	)

	if hashlen < 2 {
		return nil, nil, 0, 0
	}

	saltlen = uint8(hash[0])

	// 3 bytes of saltlen + strength + iter
	if (hashlen - 3 - int(saltlen)) <= 0 {
		return nil, nil, 0, 0
	}

	salt = hash[1 : saltlen+1]
	iter = uint8(hash[saltlen+1])

	if iter > cryptiter {
		return nil, nil, 0, 0
	}

	strength = int8(hash[saltlen+2])

	return hash[saltlen+3:], salt, iter, strength
}

func VerifyPasswordGetMeta(hashstr, pwd string) (bool, uint8, int8, uint8) {
	key, salt, iter, strength := getKeySaltIterStrength(hashstr)

	if key == nil {
		return false, 0, 0, 0
	}

	nshift, r, p := getScryptCost(strength)
	n := int(1 << nshift)

	hash, err := scrypt.Key([]byte(pwd), salt, n, r, p, len(key))

	return err == nil && bytes.Equal(hash, key), iter, strength, uint8(len(salt))
}

func VerifyPassword(hashstr, pwd string) (ok bool) {
	ok, _, _, _ = VerifyPasswordGetMeta(hashstr, pwd)
	return
}

func getScryptCost(strength int8) (uint8, int, int) {
	if strength < 1 {
		strength = 5
	} else if strength > 10 {
		strength = 10
	}

	r := 8
	if strength < 4 {
		r = 4
	} else if strength == 10 {
		r = 32
	} else if strength > 6 {
		r = 16
	}

	p := 1
	if strength == 10 {
		p = 3
	} else if strength > 6 {
		p = 2
	}

	return uint8(strength + 10), r, p
}
