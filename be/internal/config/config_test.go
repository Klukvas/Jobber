package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setMinimalEnv sets the minimum environment variables required for Load() to succeed.
func setMinimalEnv(t *testing.T) {
	t.Helper()
	t.Setenv("JWT_ACCESS_SECRET", "test-access-secret-that-is-long-enough")
	t.Setenv("JWT_REFRESH_SECRET", "test-refresh-secret-that-is-long-enough")
}

func TestLoad(t *testing.T) {
	t.Run("with valid env vars returns correct config", func(t *testing.T) {
		setMinimalEnv(t)
		t.Setenv("SERVER_PORT", "9090")
		t.Setenv("SERVER_ENV", "development")
		t.Setenv("DB_HOST", "db.example.com")
		t.Setenv("DB_PORT", "5433")
		t.Setenv("DB_MAX_CONNS", "50")
		t.Setenv("REDIS_HOST", "redis.example.com")
		t.Setenv("REDIS_PORT", "6380")
		t.Setenv("LOG_LEVEL", "debug")
		t.Setenv("JWT_ACCESS_EXPIRY", "30m")
		t.Setenv("JWT_REFRESH_EXPIRY", "336h")

		cfg, err := Load()

		require.NoError(t, err)
		assert.Equal(t, "9090", cfg.Server.Port)
		assert.Equal(t, "development", cfg.Server.Env)
		assert.Equal(t, "db.example.com", cfg.Database.Host)
		assert.Equal(t, "5433", cfg.Database.Port)
		assert.Equal(t, 50, cfg.Database.MaxConns)
		assert.Equal(t, "redis.example.com", cfg.Redis.Host)
		assert.Equal(t, "6380", cfg.Redis.Port)
		assert.Equal(t, "debug", cfg.Log.Level)
		assert.Equal(t, 30*time.Minute, cfg.JWT.AccessExpiry)
		assert.Equal(t, 336*time.Hour, cfg.JWT.RefreshExpiry)
	})

	t.Run("fails when JWT_ACCESS_SECRET is missing", func(t *testing.T) {
		t.Setenv("JWT_ACCESS_SECRET", "")
		t.Setenv("JWT_REFRESH_SECRET", "some-refresh-secret")

		_, err := Load()

		require.Error(t, err)
		assert.Contains(t, err.Error(), "JWT_ACCESS_SECRET")
	})

	t.Run("fails when JWT_REFRESH_SECRET is missing", func(t *testing.T) {
		t.Setenv("JWT_ACCESS_SECRET", "some-access-secret")
		t.Setenv("JWT_REFRESH_SECRET", "")

		_, err := Load()

		require.Error(t, err)
		assert.Contains(t, err.Error(), "JWT_REFRESH_SECRET")
	})
}

func TestLoad_ProductionGuards(t *testing.T) {
	t.Run("rejects wildcard CORS in production", func(t *testing.T) {
		t.Setenv("SERVER_ENV", "production")
		t.Setenv("ALLOWED_ORIGINS", "*")
		t.Setenv("JWT_ACCESS_SECRET", "a]very-long-secret-at-least-32-chars!!")
		t.Setenv("JWT_REFRESH_SECRET", "another-long-secret-at-least-32-chars")
		t.Setenv("DB_SSL_MODE", "require")

		_, err := Load()

		require.Error(t, err)
		assert.Contains(t, err.Error(), "ALLOWED_ORIGINS")
	})

	t.Run("rejects short JWT access secret in production", func(t *testing.T) {
		t.Setenv("SERVER_ENV", "production")
		t.Setenv("ALLOWED_ORIGINS", "https://example.com")
		t.Setenv("JWT_ACCESS_SECRET", "short")
		t.Setenv("JWT_REFRESH_SECRET", "another-long-secret-at-least-32-chars")
		t.Setenv("DB_SSL_MODE", "require")

		_, err := Load()

		require.Error(t, err)
		assert.Contains(t, err.Error(), "JWT_ACCESS_SECRET")
		assert.Contains(t, err.Error(), "32 characters")
	})

	t.Run("rejects short JWT refresh secret in production", func(t *testing.T) {
		t.Setenv("SERVER_ENV", "production")
		t.Setenv("ALLOWED_ORIGINS", "https://example.com")
		t.Setenv("JWT_ACCESS_SECRET", "a-very-long-secret-at-least-32-chars!!")
		t.Setenv("JWT_REFRESH_SECRET", "short")
		t.Setenv("DB_SSL_MODE", "require")

		_, err := Load()

		require.Error(t, err)
		assert.Contains(t, err.Error(), "JWT_REFRESH_SECRET")
		assert.Contains(t, err.Error(), "32 characters")
	})

	t.Run("rejects DB_SSL_MODE=disable in production", func(t *testing.T) {
		t.Setenv("SERVER_ENV", "production")
		t.Setenv("ALLOWED_ORIGINS", "https://example.com")
		t.Setenv("JWT_ACCESS_SECRET", "a-very-long-secret-at-least-32-chars!!")
		t.Setenv("JWT_REFRESH_SECRET", "another-long-secret-at-least-32-chars")
		t.Setenv("DB_SSL_MODE", "disable")

		_, err := Load()

		require.Error(t, err)
		assert.Contains(t, err.Error(), "DB_SSL_MODE")
	})

	t.Run("succeeds with valid production config", func(t *testing.T) {
		t.Setenv("SERVER_ENV", "production")
		t.Setenv("ALLOWED_ORIGINS", "https://example.com")
		t.Setenv("JWT_ACCESS_SECRET", "a-very-long-secret-at-least-32-chars!!")
		t.Setenv("JWT_REFRESH_SECRET", "another-long-secret-at-least-32-chars")
		t.Setenv("DB_SSL_MODE", "require")

		cfg, err := Load()

		require.NoError(t, err)
		assert.Equal(t, "production", cfg.Server.Env)
	})
}

func TestGetEnv(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		envValue     string
		defaultValue string
		expected     string
	}{
		{
			name:         "returns env value when set",
			key:          "TEST_GET_ENV_SET",
			envValue:     "custom-value",
			defaultValue: "default",
			expected:     "custom-value",
		},
		{
			name:         "returns default when env not set",
			key:          "TEST_GET_ENV_UNSET",
			envValue:     "",
			defaultValue: "fallback",
			expected:     "fallback",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				t.Setenv(tt.key, tt.envValue)
			}

			result := getEnv(tt.key, tt.defaultValue)

			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetEnvAsInt(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		envValue     string
		defaultValue int
		expected     int
	}{
		{
			name:         "parses valid integer",
			key:          "TEST_INT_VALID",
			envValue:     "42",
			defaultValue: 10,
			expected:     42,
		},
		{
			name:         "returns default for invalid integer",
			key:          "TEST_INT_INVALID",
			envValue:     "not-a-number",
			defaultValue: 10,
			expected:     10,
		},
		{
			name:         "returns default when env not set",
			key:          "TEST_INT_UNSET",
			envValue:     "",
			defaultValue: 25,
			expected:     25,
		},
		{
			name:         "parses zero",
			key:          "TEST_INT_ZERO",
			envValue:     "0",
			defaultValue: 99,
			expected:     0,
		},
		{
			name:         "parses negative number",
			key:          "TEST_INT_NEGATIVE",
			envValue:     "-5",
			defaultValue: 10,
			expected:     -5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				t.Setenv(tt.key, tt.envValue)
			}

			result := getEnvAsInt(tt.key, tt.defaultValue)

			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetEnvAsBool(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		envValue     string
		defaultValue bool
		expected     bool
	}{
		{
			name:         "parses true",
			key:          "TEST_BOOL_TRUE",
			envValue:     "true",
			defaultValue: false,
			expected:     true,
		},
		{
			name:         "parses false",
			key:          "TEST_BOOL_FALSE",
			envValue:     "false",
			defaultValue: true,
			expected:     false,
		},
		{
			name:         "parses 1 as true",
			key:          "TEST_BOOL_ONE",
			envValue:     "1",
			defaultValue: false,
			expected:     true,
		},
		{
			name:         "parses 0 as false",
			key:          "TEST_BOOL_ZERO",
			envValue:     "0",
			defaultValue: true,
			expected:     false,
		},
		{
			name:         "returns default for invalid value",
			key:          "TEST_BOOL_INVALID",
			envValue:     "maybe",
			defaultValue: true,
			expected:     true,
		},
		{
			name:         "returns default when env not set",
			key:          "TEST_BOOL_UNSET",
			envValue:     "",
			defaultValue: true,
			expected:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				t.Setenv(tt.key, tt.envValue)
			}

			result := getEnvAsBool(tt.key, tt.defaultValue)

			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDatabaseConfig_DSN(t *testing.T) {
	tests := []struct {
		name     string
		cfg      DatabaseConfig
		expected string
	}{
		{
			name: "returns correct connection string",
			cfg: DatabaseConfig{
				Host:     "localhost",
				Port:     "5432",
				User:     "jobber",
				Password: "secret",
				DBName:   "jobber_db",
				SSLMode:  "disable",
			},
			expected: "host=localhost port=5432 user=jobber password=secret dbname=jobber_db sslmode=disable",
		},
		{
			name: "with production values",
			cfg: DatabaseConfig{
				Host:     "db.prod.example.com",
				Port:     "5433",
				User:     "admin",
				Password: "p@ss!w0rd",
				DBName:   "production",
				SSLMode:  "require",
			},
			expected: "host=db.prod.example.com port=5433 user=admin password=p@ss!w0rd dbname=production sslmode=require",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.cfg.DSN()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRedisConfig_Addr(t *testing.T) {
	tests := []struct {
		name     string
		cfg      RedisConfig
		expected string
	}{
		{
			name:     "default address",
			cfg:      RedisConfig{Host: "localhost", Port: "6379"},
			expected: "localhost:6379",
		},
		{
			name:     "custom address",
			cfg:      RedisConfig{Host: "redis.example.com", Port: "6380"},
			expected: "redis.example.com:6380",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.cfg.Addr()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLoadPlansConfig(t *testing.T) {
	t.Run("loads valid YAML file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "plans.yaml")
		content := `free:
  max_jobs: 10
  max_resumes: 2
  max_applications: 5
  max_ai_requests: 3
  max_job_parses: 5
  max_resume_builders: 1
  max_cover_letters: 1
pro:
  max_jobs: 100
  max_resumes: 20
  max_applications: 50
  max_ai_requests: 50
  max_job_parses: 50
  max_resume_builders: 10
  max_cover_letters: 10
`
		err := os.WriteFile(path, []byte(content), 0644)
		require.NoError(t, err)

		plans := loadPlansConfig(path)

		require.NotNil(t, plans)
		require.Len(t, plans, 2)

		free, ok := plans["free"]
		require.True(t, ok)
		assert.Equal(t, 10, free.MaxJobs)
		assert.Equal(t, 2, free.MaxResumes)
		assert.Equal(t, 3, free.MaxAIRequests)
		assert.Equal(t, 1, free.MaxResumeBuilders)
		assert.Equal(t, 1, free.MaxCoverLetters)

		pro, ok := plans["pro"]
		require.True(t, ok)
		assert.Equal(t, 100, pro.MaxJobs)
		assert.Equal(t, 20, pro.MaxResumes)
	})

	t.Run("returns nil for invalid YAML", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "bad.yaml")
		err := os.WriteFile(path, []byte(":::invalid yaml{{{"), 0644)
		require.NoError(t, err)

		plans := loadPlansConfig(path)

		assert.Nil(t, plans)
	})

	t.Run("returns nil for missing file", func(t *testing.T) {
		plans := loadPlansConfig("/nonexistent/path/plans.yaml")

		assert.Nil(t, plans)
	})
}

func TestGetEnvAsDuration(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		envValue     string
		defaultValue time.Duration
		expected     time.Duration
	}{
		{
			name:         "parses minutes",
			key:          "TEST_DUR_MIN",
			envValue:     "30m",
			defaultValue: 15 * time.Minute,
			expected:     30 * time.Minute,
		},
		{
			name:         "parses hours",
			key:          "TEST_DUR_HOUR",
			envValue:     "2h",
			defaultValue: 1 * time.Hour,
			expected:     2 * time.Hour,
		},
		{
			name:         "parses seconds",
			key:          "TEST_DUR_SEC",
			envValue:     "45s",
			defaultValue: 10 * time.Second,
			expected:     45 * time.Second,
		},
		{
			name:         "parses complex duration",
			key:          "TEST_DUR_COMPLEX",
			envValue:     "1h30m",
			defaultValue: 1 * time.Hour,
			expected:     90 * time.Minute,
		},
		{
			name:         "returns default for invalid value",
			key:          "TEST_DUR_INVALID",
			envValue:     "not-a-duration",
			defaultValue: 5 * time.Minute,
			expected:     5 * time.Minute,
		},
		{
			name:         "returns default when env not set",
			key:          "TEST_DUR_UNSET",
			envValue:     "",
			defaultValue: 10 * time.Minute,
			expected:     10 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				t.Setenv(tt.key, tt.envValue)
			}

			result := getEnvAsDuration(tt.key, tt.defaultValue)

			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFirstOrigin(t *testing.T) {
	cases := []struct {
		name, in, want string
	}{
		{"single", "https://jobber-app.com", "https://jobber-app.com"},
		{"trailing slash trimmed", "https://jobber-app.com/", "https://jobber-app.com"},
		{"comma list picks first", "https://jobber-app.com, https://www.jobber-app.com", "https://jobber-app.com"},
		{"wildcard skipped", "*", ""},
		{"blank", "", ""},
		{"wildcard then real", "*, https://jobber-app.com", "https://jobber-app.com"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, firstOrigin(c.in))
		})
	}
}

func TestResolvePublicBaseURL(t *testing.T) {
	t.Run("explicit PUBLIC_BASE_URL wins", func(t *testing.T) {
		t.Setenv("PUBLIC_BASE_URL", "https://custom.example/")
		t.Setenv("ALLOWED_ORIGINS", "https://jobber-app.com")
		assert.Equal(t, "https://custom.example", resolvePublicBaseURL())
	})
	t.Run("derives from concrete ALLOWED_ORIGINS", func(t *testing.T) {
		t.Setenv("PUBLIC_BASE_URL", "")
		t.Setenv("ALLOWED_ORIGINS", "https://jobber-app.com, https://www.jobber-app.com")
		assert.Equal(t, "https://jobber-app.com", resolvePublicBaseURL())
	})
	t.Run("falls back to canonical when origins are wildcard", func(t *testing.T) {
		t.Setenv("PUBLIC_BASE_URL", "")
		t.Setenv("ALLOWED_ORIGINS", "*")
		assert.Equal(t, defaultPublicBaseURL, resolvePublicBaseURL())
	})
}

// The public origin is not only in og: tags any more — the password-reset email
// puts it in front of a customer as the one link they can follow. A value that
// is not an absolute http(s) URL produces a link that goes nowhere, and the
// customer has no way to tell that from a broken account. It fails the boot
// instead.
func TestLoad_PublicBaseURLGuard(t *testing.T) {
	t.Run("accepts an absolute https origin", func(t *testing.T) {
		setMinimalEnv(t)
		t.Setenv("PUBLIC_BASE_URL", "https://jobber-app.com")

		cfg, err := Load()

		require.NoError(t, err)
		assert.Equal(t, "https://jobber-app.com", cfg.Server.PublicBaseURL)
	})

	t.Run("accepts a local http origin for development", func(t *testing.T) {
		setMinimalEnv(t)
		t.Setenv("PUBLIC_BASE_URL", "http://localhost:3000")

		_, err := Load()

		require.NoError(t, err)
	})

	t.Run("refuses anything a browser could not open", func(t *testing.T) {
		for _, bad := range []string{
			"jobber-app.com",
			"/relative",
			"ftp://files.example.test",
			"https://",
			"://nonsense",
		} {
			setMinimalEnv(t)
			t.Setenv("PUBLIC_BASE_URL", bad)

			_, err := Load()

			require.Error(t, err, bad)
			assert.Contains(t, err.Error(), "PUBLIC_BASE_URL", bad)
		}
	})

	// An empty value is not a misconfiguration: it means "derive it", and the
	// derivation is already covered above.
	t.Run("leaves the derived default alone", func(t *testing.T) {
		setMinimalEnv(t)
		t.Setenv("PUBLIC_BASE_URL", "")

		cfg, err := Load()

		require.NoError(t, err)
		assert.Equal(t, defaultPublicBaseURL, cfg.Server.PublicBaseURL)
	})
}

// setPaymentsEnv turns the checkout on with every credential a valid billing
// setup needs, so each test below can knock out exactly one of them.
func setPaymentsEnv(t *testing.T) {
	t.Helper()
	setMinimalEnv(t)
	t.Setenv("FEATURE_PAYMENTS_ENABLED", "true")
	t.Setenv("FASTSPRING_API_USERNAME", "api-user")
	t.Setenv("FASTSPRING_API_PASSWORD", "api-pass")
	t.Setenv("FASTSPRING_CHECKOUT_PATH", "fluxlab/popup-jobber")
	t.Setenv("FASTSPRING_WEBHOOK_SECRET", "webhook-secret")
}

func TestLoad_BillingGuards(t *testing.T) {
	t.Run("accepts a two-segment popup checkout path", func(t *testing.T) {
		setPaymentsEnv(t)

		cfg, err := Load()

		require.NoError(t, err)
		assert.Equal(t, "fluxlab/popup-jobber", cfg.FastSpring.CheckoutPath)
		assert.True(t, cfg.Features.PaymentsEnabled)
	})

	t.Run("refuses the full-page web checkout path", func(t *testing.T) {
		// Checkout opens in the Store Builder Library popup, which can only open
		// a checkout the dashboard generated as a popup. The old Web Checkout id
		// passes every other check and then shows a real buyer an empty popup —
		// so it has to fail the boot, not the purchase.
		setPaymentsEnv(t)
		t.Setenv("FASTSPRING_CHECKOUT_PATH", "fluxlab/jobber-checkout")

		_, err := Load()

		require.Error(t, err)
		assert.Contains(t, err.Error(), "FASTSPRING_CHECKOUT_PATH")
		assert.Contains(t, err.Error(), "popup-")
	})

	t.Run("refuses any checkout id without the popup prefix", func(t *testing.T) {
		for _, path := range []string{
			"fluxlab/jobber-checkout",
			"fluxlab/checkout",
			"fluxlab/jobber-popup-checkout", // the prefix must start the segment
			"fluxlab/POPUP-jobber",          // FastSpring generates it lowercase
		} {
			t.Run(path, func(t *testing.T) {
				setPaymentsEnv(t)
				t.Setenv("FASTSPRING_CHECKOUT_PATH", path)

				_, err := Load()

				require.Error(t, err, "path %q must not boot the server", path)
			})
		}
	})

	t.Run("rejects a checkout path that is not exactly storefront-id/checkout-id", func(t *testing.T) {
		// A typo here would otherwise surface as a failed checkout for a real
		// buyer; the same whitelist the API client uses runs at startup instead.
		invalid := map[string]string{
			"single segment":   "fluxlab",
			"three segments":   "fluxlab/popup/jobber",
			"leading slash":    "/fluxlab/popup-jobber",
			"trailing slash":   "fluxlab/popup-jobber/",
			"path traversal":   "fluxlab/../accounts",
			"encoded slash":    "fluxlab/popup%2Fjobber",
			"query smuggling":  "fluxlab/popup-jobber?x=1",
			"spaces":           "flux lab/popup jobber",
			"empty segment":    "fluxlab//popup-jobber",
			"only a separator": "/",
		}

		for name, path := range invalid {
			t.Run(name, func(t *testing.T) {
				setPaymentsEnv(t)
				t.Setenv("FASTSPRING_CHECKOUT_PATH", path)

				_, err := Load()

				require.Error(t, err, "path %q must not boot the server", path)
				assert.Contains(t, err.Error(), "FASTSPRING_CHECKOUT_PATH")
			})
		}
	})

	t.Run("an unset checkout path is still refused when payments are on", func(t *testing.T) {
		setPaymentsEnv(t)
		t.Setenv("FASTSPRING_CHECKOUT_PATH", "")

		_, err := Load()

		require.Error(t, err)
		assert.Contains(t, err.Error(), "FASTSPRING_CHECKOUT_PATH")
	})

	t.Run("a bad checkout path is ignored while payments are off", func(t *testing.T) {
		setMinimalEnv(t)
		t.Setenv("FASTSPRING_CHECKOUT_PATH", "nonsense//path")

		_, err := Load()

		require.NoError(t, err, "billing config is only validated when billing is on")
	})

	t.Run("refuses to start with webhook ingestion on and no secret", func(t *testing.T) {
		// A server that boots without the secret answers every delivery with a
		// rejection: it looks healthy while dropping renewals and cancellations.
		setPaymentsEnv(t)
		t.Setenv("FASTSPRING_WEBHOOK_SECRET", "")

		_, err := Load()

		require.Error(t, err)
		assert.Contains(t, err.Error(), "FASTSPRING_WEBHOOK_SECRET")
	})

	t.Run("webhook ingestion follows payments by default", func(t *testing.T) {
		tests := []struct {
			payments string
			want     bool
		}{
			{payments: "false", want: false},
			{payments: "true", want: true},
		}

		for _, tc := range tests {
			t.Run("payments="+tc.payments, func(t *testing.T) {
				setPaymentsEnv(t)
				t.Setenv("FEATURE_PAYMENTS_ENABLED", tc.payments)

				cfg, err := Load()

				require.NoError(t, err)
				assert.Equal(t, tc.want, cfg.Features.BillingWebhookEnabled)
			})
		}
	})

	t.Run("local billing-off startup needs no billing config at all", func(t *testing.T) {
		// The default developer setup: no FastSpring credentials, no secret.
		setMinimalEnv(t)

		cfg, err := Load()

		require.NoError(t, err)
		assert.False(t, cfg.Features.PaymentsEnabled)
		assert.False(t, cfg.Features.BillingWebhookEnabled)
	})

	t.Run("webhook ingestion can outlive a closed checkout when a secret is set", func(t *testing.T) {
		// The documented kill-switch split: stop new purchases, keep recording
		// renewals and cancellations for people who already paid.
		setMinimalEnv(t)
		t.Setenv("FEATURE_PAYMENTS_ENABLED", "false")
		t.Setenv("FEATURE_BILLING_WEBHOOK_ENABLED", "true")
		t.Setenv("FASTSPRING_WEBHOOK_SECRET", "webhook-secret")

		cfg, err := Load()

		require.NoError(t, err)
		assert.False(t, cfg.Features.PaymentsEnabled)
		assert.True(t, cfg.Features.BillingWebhookEnabled)
	})

	t.Run("rejects an environment that is neither test nor live", func(t *testing.T) {
		setMinimalEnv(t)
		t.Setenv("FASTSPRING_ENVIRONMENT", "sandbox")

		_, err := Load()

		require.Error(t, err)
		assert.Contains(t, err.Error(), "FASTSPRING_ENVIRONMENT")
	})
}
