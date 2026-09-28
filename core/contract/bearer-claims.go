package contract

type BearerClaims struct {
	KYCStatus     KYCStatus `json:"kyc_status"`
	DisplayName   string    `json:"display_name"`
	Email         string    `json:"email"`
	EmailVerified bool      `json:"email_verified"`
	Photo         *string   `json:"photo"`
}
