package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
)

const (
	TokenPrefix       = "jbr_pat_"
	tokenSuffixLength = 43 // Fixed-width base62 encoding of 256 random bits.
)

func generateToken() (string, error) {
	entropy := make([]byte, 32)
	if _, err := rand.Read(entropy); err != nil {
		return "", fmt.Errorf("generate api token entropy: %w", err)
	}
	encoded := new(big.Int).SetBytes(entropy).Text(62)
	return TokenPrefix + strings.Repeat("0", tokenSuffixLength-len(encoded)) + encoded, nil
}

func hashToken(plaintext string) string {
	hash := sha256.Sum256([]byte(plaintext))
	return hex.EncodeToString(hash[:])
}

func validToken(plaintext string) bool {
	if len(plaintext) != len(TokenPrefix)+tokenSuffixLength || !strings.HasPrefix(plaintext, TokenPrefix) {
		return false
	}
	for _, char := range plaintext[len(TokenPrefix):] {
		if !(char >= '0' && char <= '9' || char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z') {
			return false
		}
	}
	return true
}
