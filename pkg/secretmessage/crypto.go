package secretmessage

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"unicode/utf8"
)

func hash(s string) string {
	hashBytes := sha256.Sum256([]byte(s))
	return hex.EncodeToString(hashBytes[:])
}

func deriveCryptoKey(key string) []byte {
	hasher := md5.New()
	hasher.Write([]byte(key))
	return hasher.Sum(nil)
}
func decrypt(input string, passphrase string) (string, error) {
	var result string
	if input == "" {
		return result, fmt.Errorf("cannot decrypt empty string")
	}
	if passphrase == "" {
		return result, fmt.Errorf("cannot decrypt with empty passphrase")
	}

	key := deriveCryptoKey(passphrase)
	ciphertext, err := hex.DecodeString(input)
	if err != nil {
		return result, err
	}
	c, err := aes.NewCipher(key)
	if err != nil {
		return result, err
	}
	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return result, err
	}
	nonceSize := gcm.NonceSize()
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return result, err
	}

	if !utf8.Valid(plaintext) {
		return result, errors.New("decryption failed")
	}
	result = string(plaintext)
	return result, nil
}

func encrypt(input string, passphrase string) (string, error) {
	return encryptWithReader(rand.Reader, input, passphrase)
}

func encryptWithReader(rr io.Reader, input string, passphrase string) (string, error) {
	var result string
	if input == "" {
		return result, fmt.Errorf("cannot encrypt empty string")
	}
	if passphrase == "" {
		return result, fmt.Errorf("cannot encrypt with empty passphrase")
	}
	key := deriveCryptoKey(passphrase)
	c, err := aes.NewCipher(key)
	if err != nil {
		return result, err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return result, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rr, nonce); err != nil {
		return result, err
	}
	ciphertext := hex.EncodeToString(gcm.Seal(nonce, nonce, []byte(input), nil))
	return ciphertext, nil
}
