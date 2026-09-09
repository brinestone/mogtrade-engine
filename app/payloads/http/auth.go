package httppayloads

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

type RotateRefreshTokenRequest struct {
	DeviceId     string `header:"x-d-id" binding:"required"`
	RefreshToken string `header:"x-refresh-token" binding:"required"`
}
