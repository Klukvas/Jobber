package http

import (
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"net/http/httptest"
	"testing"
)

func TestAIAvailabilityIsCapabilityOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, available := range []bool{false, true} {
		router := gin.New()
		router.GET("/features/ai", AIAvailabilityHandler(available))
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest("GET", "/features/ai", nil))
		require.Equal(t, 200, response.Code)
		if available {
			require.JSONEq(t, `{"available":true}`, response.Body.String())
		} else {
			require.JSONEq(t, `{"available":false}`, response.Body.String())
		}
	}
}
