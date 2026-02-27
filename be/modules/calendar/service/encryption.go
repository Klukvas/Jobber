package service

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/andreypavlenko/jobber/modules/calendar/model"
)

// Encryptor handles AES-256-GCM encryption/decryption of OAuth tokens
type Encryptor struct {
	key []byte
}

// NewEncryptor creates a new encryptor from a hex-encoded key (64 hex chars = 32 bytes)
func NewEncryptor(hexKey string) (*Encryptor, error) {
	key, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, fmt.Errorf("invalid encryption key: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("encryption key must be 32 bytes (64 hex chars), got %d bytes", len(key))
	}
	return &Encryptor{key: key}, nil
}

// Encrypt encrypts plaintext using AES-256-GCM
func (e *Encryptor) Encrypt(plaintext []byte) (ciphertext string, nonce string, err error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", "", fmt.Errorf("%w: %v", model.ErrEncryptionFailed, err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", "", fmt.Errorf("%w: %v", model.ErrEncryptionFailed, err)
	}

	nonceBytes := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonceBytes); err != nil {
		return "", "", fmt.Errorf("%w: %v", model.ErrEncryptionFailed, err)
	}

	sealed := gcm.Seal(nil, nonceBytes, plaintext, nil)
	return base64.StdEncoding.EncodeToString(sealed),
		base64.StdEncoding.EncodeToString(nonceBytes),
		nil
}

// Decrypt decrypts ciphertext using AES-256-GCM
func (e *Encryptor) Decrypt(ciphertextB64, nonceB64 string) ([]byte, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextB64)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid ciphertext encoding", model.ErrDecryptionFailed)
	}

	nonceBytes, err := base64.StdEncoding.DecodeString(nonceB64)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid nonce encoding", model.ErrDecryptionFailed)
	}

	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", model.ErrDecryptionFailed, err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", model.ErrDecryptionFailed, err)
	}

	plaintext, err := gcm.Open(nil, nonceBytes, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", model.ErrDecryptionFailed, err)
	}

	return plaintext, nil
}
