package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	httpPlatform "github.com/andreypavlenko/jobber/internal/platform/http"
	"github.com/andreypavlenko/jobber/internal/platform/telegram"
	"github.com/andreypavlenko/jobber/modules/support/model"
	"github.com/andreypavlenko/jobber/modules/support/service"
	userRepo "github.com/andreypavlenko/jobber/modules/users/repository"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func userColumns() []pgCol {
	return []pgCol{
		{"id", 25}, {"email", 25}, {"name", 25}, {"password_hash", 25},
		{"locale", 25}, {"email_verified", 16}, {"created_at", 1184}, {"updated_at", 1184},
	}
}

func userRow(id, name, email string) [][]byte {
	ts := "2023-01-02 03:04:05+00"
	return [][]byte{
		[]byte(id), []byte(email), []byte(name), []byte("hash"),
		[]byte("en"), []byte("t"), []byte(ts), []byte(ts),
	}
}

func newPool(t *testing.T, dsn string) *pgxpool.Pool {
	t.Helper()
	cfg, err := pgxpool.ParseConfig(dsn)
	require.NoError(t, err)
	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	require.NoError(t, err)
	return pool
}

// okTransport answers the Telegram API with ok:true. It is process-global, so
// tests using it must not run in parallel.
type okTransport struct{ calls int }

func (o *okTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	o.calls++
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewBufferString(`{"ok":true}`)),
		Header:     make(http.Header),
	}, nil
}

func withTelegramTransport(t *testing.T, tr http.RoundTripper) {
	t.Helper()
	orig := http.DefaultTransport
	http.DefaultTransport = tr
	t.Cleanup(func() { http.DefaultTransport = orig })
}

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func authMiddleware(userID string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	}
}

// newWorkingHandler builds a SupportHandler whose service succeeds end-to-end:
// GetByID is served by an in-process pg wire-protocol mock and Telegram by an
// in-process RoundTripper. Returns the handler and the fake pg (caller closes).
func newWorkingHandler(t *testing.T) (*SupportHandler, *fakePG) {
	fpg := newFakePG(t, userColumns(), userRow("user-1", "Jane Doe", "jane@example.com"), "")
	pool := newPool(t, fpg.dsn())
	t.Cleanup(func() { pool.Close() })
	svc := service.NewSupportService(telegram.NewClient("tok", "chat"), userRepo.NewUserRepository(pool))
	return NewSupportHandler(svc, zap.NewNop()), fpg
}

// newFailingHandler builds a SupportHandler whose service fails at user lookup
// (unreachable pool), exercising the handler's 500 error mapping.
func newFailingHandler(t *testing.T) *SupportHandler {
	pool := newPool(t, "postgres://u:p@127.0.0.1:1/db?sslmode=disable")
	t.Cleanup(func() { pool.Close() })
	svc := service.NewSupportService(telegram.NewClient("tok", "chat"), userRepo.NewUserRepository(pool))
	return NewSupportHandler(svc, zap.NewNop())
}

func doPost(router *gin.Engine, body string) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(http.MethodPost, "/support", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func decodeError(t *testing.T, w *httptest.ResponseRecorder) httpPlatform.ErrorResponse {
	t.Helper()
	var e httpPlatform.ErrorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &e))
	return e
}

func TestSupportHandler_Create_Success(t *testing.T) {
	handler, fpg := newWorkingHandler(t)
	defer fpg.Close()

	tr := &okTransport{}
	withTelegramTransport(t, tr)

	router := setupRouter()
	router.POST("/support", authMiddleware("user-1"), handler.Create)

	w := doPost(router, `{"subject":"Login broken","message":"I cannot log in at all"}`)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, tr.calls)

	var body map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "Support request sent successfully", body["message"])
}

func TestSupportHandler_Create_Unauthorized(t *testing.T) {
	handler, fpg := newWorkingHandler(t)
	defer fpg.Close()

	router := setupRouter()
	router.POST("/support", handler.Create) // no auth middleware

	w := doPost(router, `{"subject":"Login broken","message":"I cannot log in at all"}`)

	require.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Equal(t, "UNAUTHORIZED", decodeError(t, w).ErrorCode)
}

func TestSupportHandler_Create_ValidationErrors(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"malformed json", `{not json`},
		{"missing subject", `{"message":"a message long enough"}`},
		{"subject too short", `{"subject":"ab","message":"a message long enough"}`},
		{"missing message", `{"subject":"a valid subject"}`},
		{"message too short", `{"subject":"a valid subject","message":"short"}`},
		{"empty body", ``},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			handler, fpg := newWorkingHandler(t)
			defer fpg.Close()

			router := setupRouter()
			router.POST("/support", authMiddleware("user-1"), handler.Create)

			w := doPost(router, tc.body)

			require.Equal(t, http.StatusBadRequest, w.Code)
			assert.Equal(t, string(model.CodeValidationError), decodeError(t, w).ErrorCode)
		})
	}
}

func TestSupportHandler_Create_ServiceError(t *testing.T) {
	handler := newFailingHandler(t)

	router := setupRouter()
	router.POST("/support", authMiddleware("user-1"), handler.Create)

	w := doPost(router, `{"subject":"Login broken","message":"I cannot log in at all"}`)

	require.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, string(model.CodeTelegramError), decodeError(t, w).ErrorCode)
}

func TestSupportHandler_RegisterRoutes(t *testing.T) {
	handler, fpg := newWorkingHandler(t)
	defer fpg.Close()

	tr := &okTransport{}
	withTelegramTransport(t, tr)

	// Rate limiter is a pass-through middleware for the test.
	passThrough := func(c *gin.Context) { c.Next() }

	router := setupRouter()
	v1 := router.Group("/api/v1")
	handler.RegisterRoutes(v1, authMiddleware("user-1"), passThrough)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/support",
		bytes.NewBufferString(`{"subject":"Login broken","message":"I cannot log in at all"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Route is registered (not 404) and the wired handler runs successfully.
	require.NotEqual(t, http.StatusNotFound, w.Code)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestNewSupportHandler(t *testing.T) {
	svc := service.NewSupportService(telegram.NewClient("t", "c"), userRepo.NewUserRepository(nil))
	h := NewSupportHandler(svc, zap.NewNop())
	require.NotNil(t, h)
}
