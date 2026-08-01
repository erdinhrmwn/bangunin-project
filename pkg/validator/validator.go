// Package validator wraps go-playground/validator, returning field-level
// errors as a map ready to drop into pkg/response's Errors field.
package validator

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

func init() {
	_ = validate.RegisterValidation("password", func(fl validator.FieldLevel) bool {
		s := fl.Field().String()
		return len(s) >= 8 && strings.ContainsAny(s, "0123456789")
	})
}

// Struct validates v's tags and returns a field->message map, or nil if valid.
func Struct(v any) map[string]string {
	err := validate.Struct(v)
	if err == nil {
		return nil
	}

	fieldErrs, ok := err.(validator.ValidationErrors)
	if !ok {
		return map[string]string{"_error": err.Error()}
	}

	out := make(map[string]string, len(fieldErrs))
	for _, fe := range fieldErrs {
		out[fe.Field()] = message(fe)
	}
	return out
}

func message(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", fe.Field())
	case "email":
		return fmt.Sprintf("%s must be a valid email", fe.Field())
	case "min":
		return fmt.Sprintf("%s must be at least %s characters", fe.Field(), fe.Param())
	case "oneof":
		return fmt.Sprintf("%s must be one of [%s]", fe.Field(), fe.Param())
	case "password":
		return fmt.Sprintf("%s must be at least 8 characters and contain a digit", fe.Field())
	default:
		return fmt.Sprintf("%s is invalid", fe.Field())
	}
}
