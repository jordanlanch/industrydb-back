package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims represents JWT claims
type Claims struct {
	UserID int    `json:"user_id"`
	Email  string `json:"email"`
	Tier   string `json:"tier"`
	jwt.RegisteredClaims
}

// GenerateJWT generates a new JWT token
func GenerateJWT(userID int, email, tier, secret string, expirationHours int) (string, error) {
	claims := &Claims{
		UserID: userID,
		Email:  email,
		Tier:   tier,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * time.Duration(expirationHours))),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ValidateJWT validates a JWT token and returns the claims
func ValidateJWT(tokenString, secret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// ValidateJWTWithBlacklist validates a JWT token and checks if it's blacklisted
func ValidateJWTWithBlacklist(ctx context.Context, tokenString, secret string, blacklist *TokenBlacklist) (*Claims, error) {
	// First, validate the JWT signature and expiration
	claims, err := ValidateJWT(tokenString, secret)
	if err != nil {
		return nil, err
	}

	// Check if token is blacklisted
	if blacklist != nil {
		isBlacklisted, err := blacklist.IsBlacklisted(ctx, tokenString)
		if err != nil {
			return nil, fmt.Errorf("failed to check blacklist: %w", err)
		}

		if isBlacklisted {
			return nil, fmt.Errorf("token has been revoked")
		}
	}

	return claims, nil
}
