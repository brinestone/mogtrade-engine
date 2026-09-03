package encoding

type TokenEncoder interface {
	EncodeWithClaims(map[string]any) (string, error)
}

type IdGeneratorFunc func() string
