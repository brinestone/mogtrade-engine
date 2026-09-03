package auth

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"time"

	db2 "database/sql"

	enc "github.com/brinestone/mogtrade/core/encoding"
	"github.com/brinestone/mogtrade/infra/db"
)

type SignInResult struct {
	AccessToken  string `json:"accessToken" xml:"accestoken"`
	RefreshToken string `json:"refreshToken" xml:"refreshtoken"`
}

type CredentialSignInInput struct {
	Identifier           string
	Password             string
	RefreshTokenLifetime time.Duration
}

var (
	ErrNoAuthAccountFound = errors.New("account not found with provided credentials")
	ErrInavlidCredentials = errors.New("invalid credentials provided")
)

func SignInUserByCredentials(ctx context.Context, q *db.Queries, te enc.TokenEncoder, idg enc.IdGeneratorFunc, csi CredentialSignInInput) (SignInResult, error) {
	account, err := q.FindCredentialAccountById(ctx, csi.Identifier)
	if err != nil {
		if errors.Is(err, db2.ErrNoRows) {
			return SignInResult{}, ErrNoAuthAccountFound
		}
	}

	if !VerifyPassword(csi.Password, *account.Password) {
		return SignInResult{}, ErrInavlidCredentials
	}

	user, _ := q.FindUserById(ctx, account.UserID)
	accessToken, err := te.EncodeWithClaims(getUserClaims(&user))
	if err != nil {
		return SignInResult{}, err
	}

	rtSalt, _ := genSalt(20)
	refreshToken := fmt.Sprintf("%x", sha256.Sum256(append([]byte(accessToken), rtSalt...)))

	err = q.CreateRefreshToken(ctx, db.CreateRefreshTokenParams{
		ID:          idg(),
		UserID:      account.UserID,
		TokenHash:   refreshToken,
		ValidWindow: csi.RefreshTokenLifetime.String(),
	})
	if err != nil {
		return SignInResult{}, err
	}

	return SignInResult{AccessToken: accessToken, RefreshToken: refreshToken}, nil
}

func getUserClaims(u *db.User) map[string]any {
	return map[string]any{
		"display_name":   u.Name,
		"email":          u.Email,
		"email_verified": u.EmailVerified,
		"photo":          u.Image,
		"sub":            u.ID,
	}
}
