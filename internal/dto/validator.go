package dto

import (
	"fmt"
	"reflect"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/shopspring/decimal"
)

func init() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterCustomTypeFunc(func(field reflect.Value) interface{} {
			if val, ok := field.Interface().(decimal.Decimal); ok {
				f, _ := val.Float64()
				return f
			}
			return nil
		}, decimal.Decimal{})
	}
}


func FormatValidationErrors(err error) []string {
	var errs []string
	if valErrs, ok := err.(validator.ValidationErrors); ok {
		for _, fieldErr := range valErrs {
			var errMsg string
			switch fieldErr.Tag() {
			case "required":
				errMsg = fmt.Sprintf("%s is required", fieldErr.Field())
			case "email":
				errMsg = fmt.Sprintf("%s must be a valid email", fieldErr.Field())
			case "min":
				errMsg = fmt.Sprintf("%s must be at least %s characters", fieldErr.Field(), fieldErr.Param())
			case "eqfield":
				errMsg = fmt.Sprintf("%s must equal %s", fieldErr.Field(), fieldErr.Param())
			case "gt":
				errMsg = fmt.Sprintf("%s must be greater than %s", fieldErr.Field(), fieldErr.Param())
			default:
				errMsg = fmt.Sprintf("%s is invalid: %s", fieldErr.Field(), fieldErr.Tag())
			}
			errs = append(errs, errMsg)
		}
	} else {
		errs = append(errs, err.Error())
	}
	return errs
}
