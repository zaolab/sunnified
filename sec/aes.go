package sec

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
	"errors"
)

var (
	ErrNonceGenFailed = errors.New("Generating of random nonce failed")
	ErrNonceMissing   = errors.New("Unable to locate nonce in encrypted message")
)

// AesCtrEncrypt encrypts the given msg with the key using AES CTR cipher,
// and returns the encrypted slice of bytes or error if any
func AesCtrEncrypt(key []byte, msg []byte) ([]byte, error) {
	bl, _ := aes.NewCipher(padAesKey(key))
	nonce := GenRandomBytes(bl.BlockSize())

	if nonce == nil {
		return nil, ErrNonceGenFailed
	}

	st := cipher.NewCTR(bl, nonce)
	output := make([]byte, len(msg))
	st.XORKeyStream(output, msg)
	return append(nonce, output...), nil
}

// AesCtrDecrypt decrypts the given encrypted msg with the key using AES CTR cipher,
// and returns the decrypted slice of bytes or error if any
func AesCtrDecrypt(key []byte, msg []byte) ([]byte, error) {
	bl, _ := aes.NewCipher(padAesKey(key))
	blen := bl.BlockSize()

	if msglen := len(msg); msglen >= blen {
		nonce := msg[0:blen]
		msg = msg[blen:]
		st := cipher.NewCTR(bl, nonce)
		output := make([]byte, msglen-blen)
		st.XORKeyStream(output, msg)
		return output, nil
	}

	return nil, ErrNonceMissing
}

// AesGcmEncrypt encrypts the given msg with the key using AES GCM cipher,
// and returns the encrypted slice of bytes or error if any
func AesGcmEncrypt(key []byte, msg []byte) ([]byte, error) {
	bl, _ := aes.NewCipher(padAesKey(key))
	aead, err := cipher.NewGCM(bl)

	if err == nil {
		nonce := GenRandomBytes(aead.NonceSize())

		if nonce == nil {
			return nil, ErrNonceGenFailed
		}

		output := aead.Seal(nil, nonce, msg, nil)
		return append(nonce, output...), nil
	}

	return nil, err
}

// AesGcmDecrypt decrypts the given encrypted msg with the key using AES GCM cipher,
// and returns the decrypted slice of bytes or error if any
func AesGcmDecrypt(key []byte, msg []byte) ([]byte, error) {
	bl, _ := aes.NewCipher(padAesKey(key))
	aead, err := cipher.NewGCM(bl)

	if err == nil {
		nlen := aead.NonceSize()

		if msglen := len(msg); msglen > nlen {
			nonce := msg[0:nlen]
			msg = msg[nlen:]
			return aead.Open(nil, nonce, msg, nil)
		}

		return nil, ErrNonceMissing
	}

	return nil, err
}

// AesCtrEncryptBase64 encrypts the given msg with the key using AES CTR cipher,
// and returns the base64 string format of the encrypted bytes or error if any
func AesCtrEncryptBase64(key []byte, msg []byte) (b64 string, err error) {
	var res []byte
	if res, err = AesCtrEncrypt(key, msg); err == nil {
		b64 = base64.StdEncoding.EncodeToString(res)
	}
	return
}

// AesCtrDecryptBase64 decrypts the given encrypted msg of base64 format with the key using AES GCM cipher,
// and returns the decrypted slice of bytes or error if any
func AesCtrDecryptBase64(key []byte, msg string) (inp []byte, err error) {
	if inp, err = base64.StdEncoding.DecodeString(msg); err == nil {
		inp, err = AesCtrDecrypt(key, inp)
	}
	return
}

// AesGcmEncryptBase64 encrypts the given msg with the key using AES GCM cipher,
// and returns the base64 string format of the encrypted bytes or error if any
func AesGcmEncryptBase64(key []byte, msg []byte) (b64 string, err error) {
	var res []byte
	if res, err = AesGcmEncrypt(key, msg); err == nil {
		b64 = base64.StdEncoding.EncodeToString(res)
	}
	return
}

// AesGcmDecryptBase64 decrypts the given encrypted msg of base64 format with the key using AES GCM cipher,
// and returns the decrypted slice of bytes or error if any
func AesGcmDecryptBase64(key []byte, msg string) (inp []byte, err error) {
	if inp, err = base64.StdEncoding.DecodeString(msg); err == nil {
		inp, err = AesGcmDecrypt(key, inp)
	}
	return
}

func padAesKey(key []byte) []byte {
	// convert key with incorrect size to 32bytes using sha256
	if keylen := len(key); keylen != 16 && keylen != 24 && keylen != 32 {
		s256 := sha256.Sum256(key)
		key = s256[:]
	}

	return key
}
