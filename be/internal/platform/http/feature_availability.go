package http

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

// AIAvailabilityHandler exposes capability only, never provider configuration.
// Availability does not grant a plan entitlement or guarantee a request succeeds.
func AIAvailabilityHandler(available bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"available": available})
	}
}
