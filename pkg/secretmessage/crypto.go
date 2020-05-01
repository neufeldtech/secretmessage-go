package secretmessage

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"io"
)

func hash(s string) string {
	hashBytes := sha256.Sum256([]byte(s))
	return hex.EncodeToString(hashBytes[:])
}

func genKey(key string) []byte {
	hasher := md5.New()
	hasher.Write([]byte(key))
	return hasher.Sum(nil)
}
func decrypt(input string, passphrase string) (string, error) {
	var result string
	key := genKey(passphrase)
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
	return string(plaintext), nil
}

func encrypt(input string, passphrase string) (string, error) {
	var result string
	key := genKey(passphrase)
	c, err := aes.NewCipher(key)
	if err != nil {
		return result, err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return result, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return result, err
	}
	ciphertext := hex.EncodeToString(gcm.Seal(nonce, nonce, []byte(input), nil))
	return ciphertext, nil
}
