package mcp

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// The Glama directory proves who owns a connector listing by fetching this
// document next to the endpoint, with no credentials and no redirect. The
// claim value identifies the listing owner's Glama account and is meant to be
// public.
const (
	glamaClaimPath   = "/api/mcp/.well-known/glama.json"
	glamaClaimSchema = "https://glama.ai/mcp/schemas/connector.json"
	glamaClaim       = "glama_claim_HXje2uHGcFriZtK6YfUc_ckdlq_WEV1b"
)

func serveGlamaClaim(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"$schema": glamaClaimSchema, "claim": glamaClaim})
}
