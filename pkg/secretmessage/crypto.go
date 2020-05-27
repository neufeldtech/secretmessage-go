package secretmessage

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"regexp"
	"strings"
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

func decryptIV(input string, passphrase string) (string, error) {
	var result string

	re := regexp.MustCompile(`^[a-f0-9]{32}:`)
	if !re.MatchString(input) {
		return result, errors.New("input not in IV format")
	}

	input = strings.ReplaceAll(input, ":", "")

	cipherTextDecoded, err := hex.DecodeString(input)
	if err != nil {
		panic(err)
	}

	block, err := aes.NewCipher([]byte(passphrase))
	if err != nil {
		return result, err
	}

	iv := cipherTextDecoded[:aes.BlockSize]
	cipherTextBytes := []byte(cipherTextDecoded)

	plaintext := make([]byte, len(cipherTextBytes)-aes.BlockSize)
	stream := cipher.NewCTR(block, iv)
	stream.XORKeyStream(plaintext, cipherTextBytes[aes.BlockSize:])
	if !utf8.Valid(plaintext) {
		return result, errors.New("decryption failed")
	}
	result = string(plaintext)
	return result, nil
}
