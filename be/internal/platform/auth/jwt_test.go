package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJWTManager_GenerateAccessToken(t *testing.T) {
	jwtManager := NewJWTManager("access-secret-32-characters!!", "refresh-secret-32-characters!", 15*time.Minute, 7*24*time.Hour)

	t.Run("generates valid access token", func(t *testing.T) {
		userID := "user-123"

		token, err := jwtManager.GenerateAccessToken(userID)

		require.NoError(t, err)
		assert.NotEmpty(t, token)
	})

	t.Run("token contains correct user ID", func(t *testing.T) {
		userID := "user-456"

		token, err := jwtManager.GenerateAccessToken(userID)
		require.NoError(t, err)

		claims, err := jwtManager.ValidateAccessToken(token)

		require.NoError(t, err)
		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, AccessToken, claims.Type)
	})
}

func TestJWTManager_GenerateRefreshToken(t *testing.T) {
	jwtManager := NewJWTManager("access-secret-32-characters!!", "refresh-secret-32-characters!", 15*time.Minute, 7*24*time.Hour)

	t.Run("generates valid refresh token", func(t *testing.T) {
		userID := "user-123"

		token, err := jwtManager.GenerateRefreshToken(userID)

		require.NoError(t, err)
		assert.NotEmpty(t, token)
	})

	t.Run("token contains correct user ID", func(t *testing.T) {
		userID := "user-789"

		token, err := jwtManager.GenerateRefreshToken(userID)
		require.NoError(t, err)

		claims, err := jwtManager.ValidateRefreshToken(token)

		require.NoError(t, err)
		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, RefreshToken, claims.Type)
	})
}

func TestJWTManager_ValidateAccessToken(t *testing.T) {
	jwtManager := NewJWTManager("access-secret-32-characters!!", "refresh-secret-32-characters!", 15*time.Minute, 7*24*time.Hour)

	t.Run("validates valid access token", func(t *testing.T) {
		userID := "user-123"
		token, _ := jwtManager.GenerateAccessToken(userID)

		claims, err := jwtManager.ValidateAccessToken(token)

		require.NoError(t, err)
		assert.Equal(t, userID, claims.UserID)
	})

	t.Run("rejects invalid token", func(t *testing.T) {
		_, err := jwtManager.ValidateAccessToken("invalid-token")

		assert.Error(t, err)
	})

	t.Run("rejects refresh token as access token", func(t *testing.T) {
		userID := "user-123"
		refreshToken, _ := jwtManager.GenerateRefreshToken(userID)

		_, err := jwtManager.ValidateAccessToken(refreshToken)

		assert.Error(t, err)
		// Either signature validation fails or type validation fails
	})

	t.Run("rejects expired token", func(t *testing.T) {
		// Create a JWT manager with very short expiry
		shortJwt := NewJWTManager("access-secret-32-characters!!", "refresh-secret-32-characters!", -1*time.Second, 7*24*time.Hour)
		token, _ := shortJwt.GenerateAccessToken("user-123")

		_, err := jwtManager.ValidateAccessToken(token)

		assert.Error(t, err)
	})
}

func TestJWTManager_ValidateRefreshToken(t *testing.T) {
	jwtManager := NewJWTManager("access-secret-32-characters!!", "refresh-secret-32-characters!", 15*time.Minute, 7*24*time.Hour)

	t.Run("validates valid refresh token", func(t *testing.T) {
		userID := "user-123"
		token, _ := jwtManager.GenerateRefreshToken(userID)

		claims, err := jwtManager.ValidateRefreshToken(token)

		require.NoError(t, err)
		assert.Equal(t, userID, claims.UserID)
	})

	t.Run("rejects invalid token", func(t *testing.T) {
		_, err := jwtManager.ValidateRefreshToken("invalid-token")

		assert.Error(t, err)
	})

	t.Run("rejects access token as refresh token", func(t *testing.T) {
		userID := "user-123"
		accessToken, _ := jwtManager.GenerateAccessToken(userID)

		_, err := jwtManager.ValidateRefreshToken(accessToken)

		assert.Error(t, err)
		// Either signature validation fails or type validation fails
	})
}

func TestHashToken(t *testing.T) {
	t.Run("generates consistent hash", func(t *testing.T) {
		token := "test-token-12345"

		hash1 := HashToken(token)
		hash2 := HashToken(token)

		assert.Equal(t, hash1, hash2)
	})

	t.Run("generates different hashes for different tokens", func(t *testing.T) {
		token1 := "token-1"
		token2 := "token-2"

		hash1 := HashToken(token1)
		hash2 := HashToken(token2)

		assert.NotEqual(t, hash1, hash2)
	})

	t.Run("hash has expected length", func(t *testing.T) {
		token := "any-token"
		hash := HashToken(token)

		// SHA256 produces 64 hex characters
		assert.Len(t, hash, 64)
	})
}

func TestTokenType_Constants(t *testing.T) {
	assert.Equal(t, TokenType("access"), AccessToken)
	assert.Equal(t, TokenType("refresh"), RefreshToken)
}
