package middleware

import (
	"time"

	"github.com/golang-jwt/jwt/v4"
)

var jwtKey = []byte("sk") // Use a strong secret key!

type Claims struct {
	AccountID uint `json:"account_id"`
	jwt.RegisteredClaims
}

func GenerateToken(accountID uint) (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour) // Token expiration time
	claims := &Claims{
		AccountID: accountID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtKey)
}
