package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/firereach/api/internal/domain"
)

// pathID returns the :id path parameter, or answers 400 and returns false when
// it is not a UUID. Postgres rejects a malformed id with an error the
// repositories report as a server fault, so without this check a typo in a URL
// answered 500.
func pathID(c *gin.Context) (string, bool) {
	id := c.Param("id")
	if !domain.ValidID(id) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return "", false
	}
	return id, true
}
