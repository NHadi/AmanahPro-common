package models

import "github.com/dgrijalva/jwt-go"

type JWTClaims struct {
	UserID         int    `json:"user_id"`
	OrganizationId *int   `json:"organization_id"`
	Email          string `json:"email"`
	Username       string `json:"username"`
	jwt.StandardClaims
}
