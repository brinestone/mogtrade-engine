package contract

type TokenEncoder interface {
	EncodeWithClaims(BearerClaims, string) (string, error)
}

type IdConsumerFunc func(string)
type ClaimsConsumerFunc func(BearerClaims)

type TokenVerifier interface {
	VerifyToken(string, IdConsumerFunc) (bool, error)
	// VerifyWithClaims takes in a token and a pointer to an object to which the claims are going to be
	// parsed into
	VerifyWithClaims(string, ClaimsConsumerFunc) (bool, error)
}
