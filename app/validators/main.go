package validators

import (
	"regexp"

	"github.com/go-playground/validator/v10"
)

type ValidatorRegistry map[string]func(validator.FieldLevel) bool

type RequestValidator interface {
	Validators() ValidatorRegistry
}

type requestValidator struct {
}

func NewRequestValidator() RequestValidator {
	return &requestValidator{}
}

func (r requestValidator) Validators() ValidatorRegistry {
	return map[string]func(fl validator.FieldLevel) bool{
		"pattern": regexValidator(),
	}
}

func regexValidator() func(validator.FieldLevel) bool {
	return func(fl validator.FieldLevel) bool {
		pattern := fl.Param()
		regex := regexp.MustCompile(pattern)
		value := fl.Field().String()
		return regex.MatchString(value)
	}
}
