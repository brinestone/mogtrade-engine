package httppayloads

import (
	"strings"
)

type CredentialLoginRequest struct {
	Username string `json:"email" form:"email" binding:"required,email"`
	Password string `json:"password" form:"password" binding:"required"`
	DeviceId string `header:"x-d-id"`
}

func (p CredentialLoginRequest) Validate() []string {
	errors := make([]string, 0)

	if len(p.DeviceId) == 0 {
		errors = append(errors, "device id must be provided")
	}
	return errors
}

type CredentialSignUpRequest struct {
	Names           string `json:"names" form:"names" binding:"required"`
	Email           string `json:"email" form:"email" binding:"required,email"`
	Password        string `json:"password" form:"password" binding:"required,min=6,max=100"`
	ConfirmPassword string `json:"confirmPassword" form:"confirmPassword" binding:"required,eqfield=Password"`
}

func (c CredentialSignUpRequest) Initials() string {
	parts := strings.SplitN(c.Names, " ", 3)

	if len(parts) >= 2 {
		var i1, i2 string
		i1 = parts[0][0:1]
		i2 = parts[1][0:1]
		return strings.ToUpper(strings.Join([]string{i1, i2}, ""))
	} else if len(parts) == 1 {
		return strings.ToUpper(parts[0][0:1])
	}
	return ""
}

type RotateRefreshTokenRequest struct {
	DeviceId     string `header:"x-d-id" binding:"required"`
	RefreshToken string `header:"x-refresh-token" binding:"required"`
}
