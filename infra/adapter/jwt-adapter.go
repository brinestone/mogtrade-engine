package adapter

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/brinestone/mogtrade/core/contract"
	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrClaimsParsing = errors.New("could not parse JWT claims to app claims")
)

type JwtAdapter struct {
	secret        []byte
	tokenLifetime time.Duration
	audiences     []string
	issuer        string
}

// VerifyWithClaims implements [contract.TokenVerifier].
func (e *JwtAdapter) VerifyWithClaims(token string, consumer contract.ClaimsConsumerFunc) (bool, error) {
	t, err := jwt.Parse(token, func(t *jwt.Token) (any, error) {
		return e.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}), jwt.WithExpirationRequired(), jwt.WithAudience(e.audiences...))
	if err != nil {
		return false, err
	}

	j, _ := json.Marshal(t.Claims)
	var claims jwtPayload
	err = json.Unmarshal(j, &claims)
	if err != nil {
		return false, ErrClaimsParsing
	}
	defer consumer(claims.AppClaims)

	return t.Valid, nil
}

func (e *JwtAdapter) VerifyToken(token string, g contract.IdConsumerFunc) (bool, error) {
	t, err := jwt.Parse(token, func(t *jwt.Token) (any, error) {
		return e.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}), jwt.WithExpirationRequired(), jwt.WithAudience(e.audiences...))
	if err != nil {
		return false, err
	}

	id, err := t.Claims.GetSubject()
	if err == nil {
		g(id)
	}

	return t.Valid, nil
}

type jwtPayload struct {
	AppClaims contract.BearerClaims `json:"custom_fields"`
	jwt.RegisteredClaims
}

func (e *JwtAdapter) EncodeWithClaims(claims contract.BearerClaims, subject string) (string, error) {
	now := time.Now()
	regClaims := jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(now.UTC().Add(e.tokenLifetime)),
		Audience:  jwt.ClaimStrings(e.audiences),
		IssuedAt:  jwt.NewNumericDate(now.UTC()),
		Subject:   subject,
		Issuer:    e.issuer,
	}

	c := jwtPayload{
		RegisteredClaims: regClaims,
		AppClaims:        claims,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	return token.SignedString(e.secret)
}

func NewJwtTokenEncoder(secret string, lifetime time.Duration, audiences []string, issuer string) *JwtAdapter {
	return &JwtAdapter{
		[]byte(secret), lifetime, audiences, issuer,
	}
}
