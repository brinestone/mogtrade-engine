package contract

import (
	"maps"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JwtTokenEncoder struct {
	secret        []byte
	tokenLifetime time.Duration
}

func (e *JwtTokenEncoder) EncodeWithClaims(claims map[string]any) (string, error) {
	merged := make(jwt.MapClaims)
	maps.Copy(merged, claims)
	merged["exp"] = time.Now().UTC().Add(e.tokenLifetime).Unix()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, merged)
	return token.SignedString(e.secret)
}

func NewJwtTokenEncoder(secret string, lifetime time.Duration) *JwtTokenEncoder {
	return &JwtTokenEncoder{
		[]byte(secret), lifetime,
	}
}
