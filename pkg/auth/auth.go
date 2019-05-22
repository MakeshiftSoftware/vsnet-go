package auth

import (
	"errors"

	"github.com/dgrijalva/jwt-go"
)

// AccessKey access key for an authenticated user
type AccessKey struct {
	ID string
	jwt.Claims
}

// ErrInvalidToken invalid auth token
var ErrInvalidToken = errors.New("Invalid auth token")

// NewAccessKey validates auth token and returns a new access key
func NewAccessKey(auth string, jwtSecret *[]byte) (*AccessKey, error) {
	key := AccessKey{}

	verifyFunc := func(t *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	}

	token, err := jwt.ParseWithClaims(auth, &key, verifyFunc)
	if err != nil {
		return &key, err
	}

	if claims, ok := token.Claims.(*AccessKey); ok && token.Valid {
		key.ID = claims.ID
		return &key, nil
	}

	return &key, ErrInvalidToken
}
