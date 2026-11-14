package http

import (
	"net/http"
	"strings"

	"github.com/andreypavlenko/jobber/internal/platform/ids"
	"github.com/gin-gonic/gin"
)

// CanonicalUUIDParams rejects a request whose UUID path parameters are
// malformed and rewrites the valid ones to canonical form, so handlers can pass
// c.Param straight to storage. Without it a malformed id reaches Postgres as a
// 500, and spellings uuid.Parse accepts but Postgres rejects (`urn:uuid:…`) do
// the same. Install it on the engine before any route is registered.
func CanonicalUUIDParams() gin.HandlerFunc {
	return func(c *gin.Context) {
		canonical, invalid := canonicalParams(c.Params)
		if invalid != "" {
			RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid "+invalid)
			c.Abort()
			return
		}
		c.Params = canonical
		c.Next()
	}
}

// nonUUIDParams are the route parameters that legitimately carry something
// other than a UUID.
var nonUUIDParams = map[string]bool{"token": true, "any": true}

// isUUIDParam reports whether a route parameter carries a UUID by naming
// convention: `:id`, any `:…Id`, and the admin `:grant`. Everything else
// (such as the public share `:token`) passes through untouched.
func isUUIDParam(name string) bool {
	return name == "id" || name == "grant" || strings.HasSuffix(name, "Id")
}

// canonicalParams returns a copy of params with every UUID parameter in
// canonical form, or the name of the first one that is not a UUID.
func canonicalParams(params gin.Params) (gin.Params, string) {
	canonical := make(gin.Params, len(params))
	for i, param := range params {
		canonical[i] = param
		if !isUUIDParam(param.Key) {
			continue
		}
		value, ok := ids.CanonicalUUID(param.Value)
		if !ok {
			return nil, param.Key
		}
		canonical[i].Value = value
	}
	return canonical, ""
}

// UnclassifiedPathParams lists routes whose parameters neither follow the UUID
// naming convention nor are known non-UUID params. Such a route would bypass
// CanonicalUUIDParams silently (`:jobID`, `:uuid`), so startup refuses it.
func UnclassifiedPathParams(routes gin.RoutesInfo) []string {
	var unclassified []string
	for _, route := range routes {
		for _, segment := range strings.Split(route.Path, "/") {
			if len(segment) < 2 || (segment[0] != ':' && segment[0] != '*') {
				continue
			}
			name := segment[1:]
			if !isUUIDParam(name) && !nonUUIDParams[name] {
				unclassified = append(unclassified, route.Method+" "+route.Path)
			}
		}
	}
	return unclassified
}
