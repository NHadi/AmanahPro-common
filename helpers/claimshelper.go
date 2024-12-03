package helpers

import (
	"fmt"
	"net/http"

	jwtModels "github.com/NHadi/AmanahPro-common/models"
	"github.com/gin-gonic/gin"
)

// GetClaims extracts and returns the JWT claims from the context.
func GetClaims(c *gin.Context) (*jwtModels.JWTClaims, error) {
	// Extract user claims from context
	userClaims, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return nil, fmt.Errorf("unauthorized")
	}

	claims, ok := userClaims.(*jwtModels.JWTClaims)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}
