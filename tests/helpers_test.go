package tests

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func generateTestJWT(adminID, secret string) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": adminID,
		"exp": time.Now().Add(1 * time.Hour).Unix(),
		"iat": time.Now().Unix(),
	})
	tokenString, _ := token.SignedString([]byte(secret))
	return tokenString
}
