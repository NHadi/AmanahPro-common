package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/NHadi/AmanahPro-common/models"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func JWTAuthMiddleware(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		logrus.Infof("Authorization header received: %s", authHeader)

		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			logrus.Warn("Unauthorized access: Missing or malformed Authorization header")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: Invalid token format"})
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims := &models.JWTClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				logrus.Errorf("Unexpected signing method: %v", token.Header["alg"])
				return nil, fmt.Errorf("unexpected signing method")
			}
			return []byte(jwtSecret), nil
		})

		if err != nil || !token.Valid {
			logrus.WithError(err).Warn("Invalid JWT token")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: Invalid token"})
			c.Abort()
			return
		}

		if claims.UserID <= 0 {
			logrus.Warn("Invalid claims: UserID is missing or invalid")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: Invalid claims"})
			c.Abort()
			return
		}

		logrus.Infof("User authenticated: %s", claims.Username)
		c.Set("user", claims)
		c.Next()
	}
}
