package bcrypt

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
)

const DefaultCost = 10

func GenerateFromPassword(password []byte, cost int) ([]byte, error) {
	sum := sha256.Sum256(password)
	return []byte("fakebcrypt$" + hex.EncodeToString(sum[:])), nil
}

func CompareHashAndPassword(hashedPassword []byte, password []byte) error {
	expected, _ := GenerateFromPassword(password, DefaultCost)
	if subtle.ConstantTimeCompare(expected, hashedPassword) != 1 {
		return errors.New("password mismatch")
	}
	return nil
}
