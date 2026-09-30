package contract

import "fmt"

type Gender string

const (
	GenderMale      Gender = "male"
	GenderFemale    Gender = "female"
	GenderUndefined Gender = "undefined"
)

func (e *Gender) Scan(src any) error {
	switch s := src.(type) {
	case []byte:
		*e = Gender(s)
	case string:
		*e = Gender(s)
	default:
		return fmt.Errorf("unsupported scan type for Gender: %T", src)
	}
	return nil
}
