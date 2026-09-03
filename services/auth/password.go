package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"

	"golang.org/x/crypto/argon2"
)

const (
	saltLen = 16
	keyLen  = 32
	ttime   = 1
	mem     = 64 << 10 // 64 MB
	threads = 4
)

func compareHash(a, b []byte) bool {
	return subtle.ConstantTimeCompare(a, b) == 1
}

func VerifyPassword(pass string, existingHash string) bool {
	hashB, err := hex.DecodeString(existingHash)
	if err != nil {
		return false
	}

	// Validate length to prevent out-of-bounds panics
	expectedLen := saltLen + keyLen
	if len(hashB) != expectedLen {
		return false
	}

	salt := hashB[0:saltLen]
	storedKey := hashB[saltLen:]

	passB := []byte(pass)
	newHash := argon2.IDKey(passB, salt, ttime, mem, threads, keyLen)

	// Compare the newly generated key against the stored key portion
	return compareHash(newHash, storedKey)
}

func HashPassword(pass string) (string, error) {
	salt, err := genSalt(saltLen)
	if err != nil {
		return "", err
	}
	hashed := argon2.IDKey([]byte(pass), salt, ttime, mem, threads, keyLen)
	hashed = append(salt, hashed...)
	return hex.EncodeToString(hashed), nil
}

func genSalt(len int) ([]byte, error) {
	salt := make([]byte, len)
	_, err := rand.Read(salt)
	if err != nil {
		return nil, err
	}
	return salt, nil
}
