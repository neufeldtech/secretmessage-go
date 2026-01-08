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
	"regexp"
	"unicode/utf8"

	"golang.org/x/crypto/argon2"
)

// v2$<salt>$<ciphertext>
// |    |        |
// |    |        hex-encoded encrypted payload
// |    |
// |    salt rand.Text()
// |
// version identifier
var V2EncryptionRegExp = regexp.MustCompile(`^v2\$([a-zA-Z0-9]+)\$([a-fA-F0-9]+)$`)

const Pepper string = "XP47CYQ5SWBR2ZUFPUFROF55PR"

// secureSecretID takes a string and returns a deterministic argon2 hash with a fixed system-wide salt
// to be used for generating the hashed secretID we store in the database
func secureSecretID(input string) string {
	return hex.EncodeToString(argon2.IDKey([]byte(input), []byte(Pepper), 1, 64*1024, 1, 32))
}

// hash returns a sha256 hash of a given input in hex encoding
func hash(s string) string {
	hashBytes := sha256.Sum256([]byte(s))
	return hex.EncodeToString(hashBytes[:])
}

// deriveCryptoKey takes a passphrase and returns a 16 byte slice which can be passed as an input to aes encrypt
func deriveCryptoKey(passphrase string) []byte {
	hasher := md5.New()
	hasher.Write([]byte(passphrase))
	return hasher.Sum(nil)
}

// deriveCryptoKeyV2 takes a passphrase and returns a 32 byte slice which can be used as an encryption key
func deriveCryptoKeyV2(passphrase string, salt string) []byte {
	return argon2.IDKey([]byte(passphrase), []byte(salt), 1, 64*1024, 1, 32)
}

// decrypt takes a hex-encoded input and passphrase and returns a decrypted string in plaintext
func decrypt(input string, passphrase string) (string, error) {
	var result string
	if input == "" {
		return result, fmt.Errorf("cannot decrypt empty string")
	}
	if passphrase == "" {
		return result, fmt.Errorf("cannot decrypt with empty passphrase")
	}

	var key []byte

	// If our encrypted string was prefaced with v2$ that means we encrypted with new v2 format
	if V2EncryptionRegExp.MatchString(input) {
		matches := V2EncryptionRegExp.FindStringSubmatch(input)
		// extract the salt from first capture group
		salt := matches[1]
		// actual payload to decrypt (v2 prefix and salt removed)
		input = matches[2]
		key = deriveCryptoKeyV2(passphrase, salt)
	} else {
		key = deriveCryptoKey(passphrase)
	}

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

// encrypt wraps encryptWithReader
func encrypt(input string, passphrase string) (string, error) {
	return encryptWithReader(rand.Reader, input, passphrase)
}

// encryptWithReader takes an input string and passphrase and returns a hex-encoded encrypted string
func encryptWithReader(rr io.Reader, input string, passphrase string) (string, error) {
	var result string
	if input == "" {
		return result, fmt.Errorf("cannot encrypt empty string")
	}
	if passphrase == "" {
		return result, fmt.Errorf("cannot encrypt with empty passphrase")
	}
	salt := rand.Text()
	key := deriveCryptoKeyV2(passphrase, salt)
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
	return fmt.Sprintf("v2$%s$%s", salt, ciphertext), nil
}
