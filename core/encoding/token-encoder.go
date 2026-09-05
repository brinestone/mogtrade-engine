package encoding

type TokenEncoder interface {
	EncodeWithClaims(map[string]any, string) (string, error)
}

type IdCallbackFunc func(string)
type TokenVerifier interface {
	VerifyToken(string, IdCallbackFunc) (bool, error)
}

type IdGeneratorFunc func() string
