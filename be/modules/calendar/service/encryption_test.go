package service

import (
	"crypto/rand"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func generateTestKey() string {
	key := make([]byte, 32)
	_, _ = rand.Read(key)
	return hex.EncodeToString(key)
}

func TestEncryptor_RoundTrip(t *testing.T) {
	hexKey := generateTestKey()
	enc, err := NewEncryptor(hexKey)
	require.NoError(t, err)

	plaintext := []byte(`{"access_token":"ya29.test","refresh_token":"1//test","expiry":"2025-01-01T00:00:00Z"}`)

	ciphertext, nonce, err := enc.Encrypt(plaintext)
	require.NoError(t, err)
	assert.NotEmpty(t, ciphertext)
	assert.NotEmpty(t, nonce)
	assert.NotEqual(t, string(plaintext), ciphertext)

	decrypted, err := enc.Decrypt(ciphertext, nonce)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestEncryptor_DifferentNonces(t *testing.T) {
	hexKey := generateTestKey()
	enc, err := NewEncryptor(hexKey)
	require.NoError(t, err)

	plaintext := []byte("test data")

	ct1, nonce1, err := enc.Encrypt(plaintext)
	require.NoError(t, err)

	ct2, nonce2, err := enc.Encrypt(plaintext)
	require.NoError(t, err)

	// Same plaintext should produce different ciphertexts due to random nonces
	assert.NotEqual(t, ct1, ct2)
	assert.NotEqual(t, nonce1, nonce2)
}

func TestEncryptor_WrongKey(t *testing.T) {
	key1 := generateTestKey()
	key2 := generateTestKey()

	enc1, err := NewEncryptor(key1)
	require.NoError(t, err)

	enc2, err := NewEncryptor(key2)
	require.NoError(t, err)

	plaintext := []byte("secret data")
	ciphertext, nonce, err := enc1.Encrypt(plaintext)
	require.NoError(t, err)

	_, err = enc2.Decrypt(ciphertext, nonce)
	assert.Error(t, err)
}

func TestEncryptor_TamperedCiphertext(t *testing.T) {
	hexKey := generateTestKey()
	enc, err := NewEncryptor(hexKey)
	require.NoError(t, err)

	plaintext := []byte("important data")
	ciphertext, nonce, err := enc.Encrypt(plaintext)
	require.NoError(t, err)

	// Tamper with ciphertext
	tampered := ciphertext[:len(ciphertext)-2] + "xx"
	_, err = enc.Decrypt(tampered, nonce)
	assert.Error(t, err)
}

func TestNewEncryptor_InvalidKey(t *testing.T) {
	t.Run("invalid hex", func(t *testing.T) {
		_, err := NewEncryptor("not-hex")
		assert.Error(t, err)
	})

	t.Run("wrong length", func(t *testing.T) {
		_, err := NewEncryptor("aabbccdd") // only 4 bytes
		assert.Error(t, err)
	})

	t.Run("valid key", func(t *testing.T) {
		key := generateTestKey()
		enc, err := NewEncryptor(key)
		assert.NoError(t, err)
		assert.NotNil(t, enc)
	})
}
