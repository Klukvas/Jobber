package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashPassword(t *testing.T) {
	t.Run("hashes password successfully", func(t *testing.T) {
		password := "securePassword123"

		hash, err := HashPassword(password)

		require.NoError(t, err)
		assert.NotEmpty(t, hash)
		assert.NotEqual(t, password, hash)
	})

	t.Run("generates different hashes for same password", func(t *testing.T) {
		password := "securePassword123"

		hash1, err1 := HashPassword(password)
		hash2, err2 := HashPassword(password)

		require.NoError(t, err1)
		require.NoError(t, err2)
		// bcrypt uses random salt, so hashes should be different
		assert.NotEqual(t, hash1, hash2)
	})

	t.Run("handles empty password", func(t *testing.T) {
		hash, err := HashPassword("")

		require.NoError(t, err)
		assert.NotEmpty(t, hash)
	})

	t.Run("handles long password", func(t *testing.T) {
		// bcrypt has a 72 byte limit - this test verifies behavior
		longPassword := string(make([]byte, 100))

		hash, err := HashPassword(longPassword)

		// bcrypt truncates at 72 bytes, so this should still work
		// In newer versions of bcrypt library, it may return an error
		if err != nil {
			assert.Contains(t, err.Error(), "72 bytes")
		} else {
			assert.NotEmpty(t, hash)
		}
	})
}

func TestVerifyPassword(t *testing.T) {
	t.Run("verifies correct password", func(t *testing.T) {
		password := "securePassword123"
		hash, _ := HashPassword(password)

		err := VerifyPassword(password, hash)

		assert.NoError(t, err)
	})

	t.Run("rejects incorrect password", func(t *testing.T) {
		password := "securePassword123"
		wrongPassword := "wrongPassword456"
		hash, _ := HashPassword(password)

		err := VerifyPassword(wrongPassword, hash)

		assert.Error(t, err)
	})

	t.Run("rejects empty password against non-empty hash", func(t *testing.T) {
		password := "securePassword123"
		hash, _ := HashPassword(password)

		err := VerifyPassword("", hash)

		assert.Error(t, err)
	})

	t.Run("handles malformed hash", func(t *testing.T) {
		err := VerifyPassword("password", "invalid-hash")

		assert.Error(t, err)
	})

	t.Run("verifies empty password with empty password hash", func(t *testing.T) {
		hash, _ := HashPassword("")

		err := VerifyPassword("", hash)

		assert.NoError(t, err)
	})
}

func TestDefaultCost(t *testing.T) {
	assert.Equal(t, 12, DefaultCost)
}

func BenchmarkHashPassword(b *testing.B) {
	password := "testPassword123"
	for i := 0; i < b.N; i++ {
		HashPassword(password)
	}
}

func BenchmarkVerifyPassword(b *testing.B) {
	password := "testPassword123"
	hash, _ := HashPassword(password)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		VerifyPassword(password, hash)
	}
}
