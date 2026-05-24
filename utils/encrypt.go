package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/md5"
	crand "crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"io"
	"regexp"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

var (
	legacySha256Pattern = regexp.MustCompile(`^[a-f0-9]{64}$`)
	md5HashPattern      = regexp.MustCompile(`^[a-f0-9]{32}$`)
)

func Sha2Encrypt(raw string) string {
	// return 64-bit hash
	hash := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(hash[:])
}

func HmacSha256(raw string, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(raw))
	return hex.EncodeToString(mac.Sum(nil))
}

func HashPassword(raw string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(raw), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	return string(hash), nil
}

func VerifyPassword(raw string, stored string) (bool, bool) {
	if err := bcrypt.CompareHashAndPassword([]byte(stored), []byte(raw)); err == nil {
		return true, false
	}

	if legacySha256Pattern.MatchString(stored) && Sha2Encrypt(raw) == stored {
		return true, true
	}

	return false, false
}

func IsSha256Hash(raw string) bool {
	return legacySha256Pattern.MatchString(strings.TrimSpace(raw))
}

func IsMd5Hash(raw string) bool {
	return md5HashPattern.MatchString(strings.TrimSpace(raw))
}

func Sha2EncryptForm(form interface{}) string {
	// return 64-bit hash
	hash := sha256.Sum256([]byte(ToJson(form)))
	return hex.EncodeToString(hash[:])
}

func Base64Encode(raw string) string {
	return base64.StdEncoding.EncodeToString([]byte(raw))
}

func Base64EncodeBytes(raw []byte) string {
	return base64.StdEncoding.EncodeToString(raw)
}

func Base64Decode(raw string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(raw)
}

func Base64DecodeBytes(raw string) []byte {
	if data, err := base64.StdEncoding.DecodeString(raw); err == nil {
		return data
	} else {
		return []byte{}
	}
}

func Md5Encrypt(raw string) string {
	// return 32-bit hash
	hash := md5.Sum([]byte(raw))
	return hex.EncodeToString(hash[:])
}

func Md5EncryptForm(form interface{}) string {
	// return 32-bit hash
	hash := md5.Sum([]byte(ToJson(form)))
	return hex.EncodeToString(hash[:])
}

func AES256Encrypt(key string, data string) (string, error) {
	text := []byte(data)
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}

	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(crand.Reader, iv); err != nil {
		return "", err
	}

	encryptor := cipher.NewCFBEncrypter(block, iv)

	ciphertext := make([]byte, len(text))
	encryptor.XORKeyStream(ciphertext, text)
	return hex.EncodeToString(ciphertext), nil
}

func AES256Decrypt(key string, data string) (string, error) {
	ciphertext, err := hex.DecodeString(data)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}

	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	decryptor := cipher.NewCFBDecrypter(block, iv)
	plaintext := make([]byte, len(ciphertext))
	decryptor.XORKeyStream(plaintext, ciphertext)

	return string(plaintext), nil
}
