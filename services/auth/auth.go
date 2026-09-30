package auth

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"time"

	"database/sql"

	"github.com/brinestone/mogtrade/core/contract"
	"github.com/brinestone/mogtrade/infra/db"
	"github.com/brinestone/mogtrade/services/kyc"
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
	RefreshTokenId       string
	Identifier           string
	Password             string
	DeviceId             string
	RefreshTokenLifetime time.Duration
}

type CredentialSignUpInput struct {
	Name       string
	Identifier string
	Password   string
	Email      string
	Photo      string
}

type RotateAccessTokenInput struct {
	RefreshTokenId string
	DeviceId       string
	Hash           string
	Lifetime       time.Duration
}

var (
	ErrNoAuthAccountFound   = errors.New("account not found with provided credentials")
	ErrInavlidCredentials   = errors.New("invalid credentials provided")
	ErrAccountAlreadyExists = errors.New("an account with the provided credentials already exists")
	ErrRefreshTokenUnusable = errors.New("the provided refreshtoken has been revoked or expired")
	ErrUserNotFound         = errors.New("user not found")
	ErrRefreshTokenNotFound = errors.New("refresh token not found")
)

func UserExistsWithId(ctx context.Context, q *db.Queries, id string) (bool, error) {
	return q.UserExistsWithId(ctx, id)
}

func RotateAccessToken(ctx context.Context, q *db.Queries, te contract.TokenEncoder, r RotateAccessTokenInput) (SignInResult, error) {
	row, err := q.LookupRefreshTokenByDevice(ctx, db.LookupRefreshTokenByDeviceParams{DeviceID: r.DeviceId, TokenHash: r.Hash})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return SignInResult{}, ErrRefreshTokenNotFound
		}
		return SignInResult{}, err
	}

	if row.Usable == nil || !*row.Usable {
		return SignInResult{}, ErrRefreshTokenUnusable
	}

	user, err := q.FindUserById(ctx, row.UserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return SignInResult{}, ErrUserNotFound
		}
		return SignInResult{}, err
	}

	var kycStatus contract.KYCStatus = contract.KYCPending
	if k, err := kyc.GetByUser(ctx, q, user.ID); err == nil {
		kycStatus = k.Status
	} else if !errors.Is(err, kyc.ErrKycNotFound) {
		return SignInResult{}, err
	}
	newAccessToken, newRefresh, err := generateAuthTokenPairs(user, kycStatus, te)
	if err != nil {
		return SignInResult{}, err
	}
	println(kycStatus)
	q.InvalidateRefreshTokensForDevice(ctx, r.DeviceId)
	err = q.CreateRefreshToken(ctx, db.CreateRefreshTokenParams{
		ID:          r.RefreshTokenId,
		UserID:      row.UserID,
		DeviceID:    r.DeviceId,
		TokenHash:   newRefresh,
		ValidWindow: r.Lifetime.String(),
	})
	if err != nil {
		return SignInResult{}, err
	}

	return SignInResult{AccessToken: newAccessToken, RefreshToken: newRefresh}, nil
}

func SignUpUserByCredentials(ctx context.Context, q *db.Queries, idg contract.IdGeneratorFunc, csi CredentialSignUpInput) (SignUpResult, error) {
	exists, err := q.CredentialAccountExistsByIdentifier(ctx, csi.Identifier)
	if err != nil {
		return SignUpResult{}, err
	}

	if exists {
		return SignUpResult{}, ErrAccountAlreadyExists
	}

	userId := idg()
	accountId := idg()
	var imagePtr *string = &csi.Photo
	if len(csi.Photo) == 0 {
		imagePtr = nil
	}
	timestamp, err := q.CreateUser(ctx, db.CreateUserParams{
		ID:    userId,
		Name:  csi.Name,
		Email: csi.Email,
		Image: imagePtr,
	})
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

func SignInUserByCredentials(ctx context.Context, q *db.Queries, te contract.TokenEncoder, csi CredentialSignInInput) (SignInResult, error) {
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
	var kycStatus contract.KYCStatus = contract.KYCPending
	if k, err := kyc.GetByUser(ctx, q, user.ID); err == nil {
		kycStatus = k.Status
	} else if !errors.Is(err, kyc.ErrKycNotFound) {
		return SignInResult{}, err
	}
	accessToken, refreshToken, err := generateAuthTokenPairs(user, kycStatus, te)
	if err != nil {
		return SignInResult{}, err
	}
	q.InvalidateRefreshTokensForDevice(ctx, csi.DeviceId)
	err = q.CreateRefreshToken(ctx, db.CreateRefreshTokenParams{
		ID:          csi.RefreshTokenId,
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

func generateAuthTokenPairs(u db.User, kycStatus contract.KYCStatus, te contract.TokenEncoder) (string, string, error) {
	claims := getUserClaims(&u, kycStatus)
	accessToken, err := te.EncodeWithClaims(claims, u.ID)
	if err != nil {
		return "", "", err
	}
	rtSalt, _ := genSalt(20)
	refreshToken := fmt.Sprintf("%x", sha256.Sum256(append([]byte(accessToken), rtSalt...)))
	return accessToken, refreshToken, nil
}

func getUserClaims(u *db.User, kycStatus contract.KYCStatus) contract.BearerClaims {
	return contract.BearerClaims{
		KYCStatus:     kycStatus,
		DisplayName:   u.Name,
		Email:         u.Email,
		EmailVerified: u.EmailVerified,
		Photo:         u.Image,
	}
}
