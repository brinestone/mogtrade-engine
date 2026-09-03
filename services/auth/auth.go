package auth

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"time"

	"database/sql"

	enc "github.com/brinestone/mogtrade/core/encoding"
	"github.com/brinestone/mogtrade/infra/db"
)

type SignInResult struct {
	AccessToken  string `json:"accessToken" xml:"accestoken"`
	RefreshToken string `json:"refreshToken" xml:"refreshtoken"`
}

type SignUpResult struct {
	Timestamp       time.Time          `json:"timestamp" xml:"timestamp"`
	UserId          string             `json:"userId" xml:"user_id"`
	AccountProvider db.AccountProvider `json:"accountProvider" xml:"account_provider"`
	AccountId       string             `json:"accountId" xml:"account-id"`
}

type CredentialSignInInput struct {
	Identifier           string
	Password             string
	RefreshTokenLifetime time.Duration
	DeviceId             string
}

type CredentialSignUpInput struct {
	Name       string
	Identifier string
	Password   string
	Email      string
}

var (
	ErrNoAuthAccountFound   = errors.New("account not found with provided credentials")
	ErrInavlidCredentials   = errors.New("invalid credentials provided")
	ErrAccountAlreadyExists = errors.New("an account with the provided credentials already exists")
)

func SignUpUserByCredentials(ctx context.Context, q *db.Queries, idg enc.IdGeneratorFunc, csi CredentialSignUpInput) (SignUpResult, error) {
	exists, err := q.CredentialAccountExistsByIdentifier(ctx, csi.Identifier)
	if err != nil {
		return SignUpResult{}, err
	}

	if exists {
		return SignUpResult{}, ErrAccountAlreadyExists
	}

	userId := idg()
	accountId := idg()
	timestamp, err := q.CreateUser(ctx, db.CreateUserParams{ID: userId, Name: csi.Name, Email: csi.Email})
	if err != nil {
		return SignUpResult{}, err
	}

	hash, err := HashPassword(csi.Password)
	if err != nil {
		return SignUpResult{}, err
	}
	_, err = q.CreateCredentialAccount(ctx, db.CreateCredentialAccountParams{ID: accountId, AccountID: csi.Identifier, Password: &hash, UserID: userId})

	return SignUpResult{
		Timestamp:       timestamp.Time,
		UserId:          userId,
		AccountProvider: db.AccountProviderCredential,
		AccountId:       accountId,
	}, nil
}

func SignInUserByCredentials(ctx context.Context, q *db.Queries, te enc.TokenEncoder, idg enc.IdGeneratorFunc, csi CredentialSignInInput) (SignInResult, error) {
	account, err := q.FindCredentialAccountById(ctx, csi.Identifier)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
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

	q.InvalidateRefreshTokensForDevice(ctx, csi.DeviceId)
	err = q.CreateRefreshToken(ctx, db.CreateRefreshTokenParams{
		ID:          idg(),
		UserID:      account.UserID,
		DeviceID:    csi.DeviceId,
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
