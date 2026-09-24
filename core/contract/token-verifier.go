package contract

type TokenEncoder interface {
	EncodeWithClaims(map[string]any, string) (string, error)
}

type IdConsumerFunc func(string)
type ClaimsConsumerFunc func(map[string]any)

type TokenVerifier interface {
	VerifyToken(string, IdConsumerFunc) (bool, error)
	// VerifyWithClaims takes in a token and a pointer to an object to which the claims are going to be
	// parsed into
	VerifyWithClaims(string, ClaimsConsumerFunc) (bool, error)
}
