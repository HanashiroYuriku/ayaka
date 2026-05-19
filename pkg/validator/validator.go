package validator

import (
	"be-ayaka/internal/core/customerrors"
	"context"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"unicode"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
	"gorm.io/gorm"
)

// Validator is an interface for validating data
type Validator interface {
	Validate(ctx context.Context, data interface{}) error
}

// GoValidator is a struct that implements the Validator interface using go-playground/validator
type GoValidator struct {
	validate *validator.Validate
	uni      ut.Translator
	DB       *gorm.DB
}

// ValidationError is a custom error type for validation errors
type ValidationError struct {
	ErrorFields map[string]string `json:"errorFields,omitempty"`
	IsConflict  bool              `json:"-"`
	IsNotFound  bool              `json:"-"`
}

var (
	usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_.-]+$`)
)

// NewGoValidator creates a new instance of GoValidator with custom validators and messages
func NewGoValidator(db *gorm.DB) *GoValidator {
	v := validator.New()
	eng := en.New()
	uni := ut.New(eng, eng)
	trans, _ := uni.GetTranslator("en")

	en_translations.RegisterDefaultTranslations(v, trans)

	// register custom tag name function to use json tag instead of struct field name
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})

	gv := &GoValidator{validate: v, uni: trans, DB: db}
	gv.registerCustomValidators()
	gv.registerCustomMessages(trans)

	return gv
}

// registerCustomMessages registers custom error messages for validation tags
func (v *GoValidator) registerCustomMessages(trans ut.Translator) {
	// Override Required
	v.validate.RegisterTranslation("required", trans, func(ut ut.Translator) error {
		return ut.Add("required", "{0} is a required field", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		return fmt.Sprintf("%s is a required field", toProperCase(fe.Field()))
	})

	// Override Email
	v.validate.RegisterTranslation("email", trans, func(ut ut.Translator) error {
		return ut.Add("email", "{0} must be a valid email address", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		return fmt.Sprintf("%s must be a valid email address", toProperCase(fe.Field()))
	})

	// Override Min
	v.validate.RegisterTranslation("min", trans, func(ut ut.Translator) error {
		return ut.Add("min", "{0} must be at least {1} characters long", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		return fmt.Sprintf("%s must be at least %s characters long", toProperCase(fe.Field()), fe.Param())
	})
}

// registerCustomValidators registers custom validation functions and their corresponding error messages
func (v *GoValidator) registerCustomValidators() {
	v.validate.RegisterValidation("unique", v.uniqueValidator)
	v.validate.RegisterValidation("incolumn", v.incolumnValidator)
	v.validate.RegisterValidation("complexpassword", v.complexPasswordValidator)
	v.validate.RegisterValidation("whitespace", v.whiteSpaceValidator)
	v.validate.RegisterValidation("username", v.usernameValidator)

	// unique
	v.validate.RegisterTranslation("unique", v.uni, func(ut ut.Translator) error {
		return ut.Add("unique", "{0} already exists", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		return fmt.Sprintf("%s already exists", fe.Value().(string))
	})

	// incolumn
	v.validate.RegisterTranslation("incolumn", v.uni, func(ut ut.Translator) error {
		return ut.Add("incolumn", "{0} not found", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		return fmt.Sprintf("%s does not exists", fe.Value().(string))
	})

	// complexpassword
	v.validate.RegisterTranslation("complexpassword", v.uni, func(ut ut.Translator) error {
		return ut.Add("complexpassword", "{0} not valid", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		return "Password must be 8-64 characters long, contain: uppercase, lowercase, special character and number"
	})

	// username
	v.validate.RegisterTranslation("username", v.uni, func(ut ut.Translator) error {
		return ut.Add("username", "{0} can only contain letters, numbers, underscores, hyphens, and dots", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		return fmt.Sprintf("%s can only contain letters, numbers, underscores, hyphens, and dots", toProperCase(fe.Field()))
	})

	// whitespace
	v.validate.RegisterTranslation("whitespace", v.uni, func(ut ut.Translator) error {
		return ut.Add("whitespace", "{0} cannot contain spaces", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		return fmt.Sprintf("%s cannot contain spaces", toProperCase(fe.Field()))
	})
}

// Custom validation functions
func (v *GoValidator) uniqueValidator(fl validator.FieldLevel) bool {
	params := strings.Split(fl.Param(), "->")
	if len(params) != 2 {
		return false
	}

	tableName := params[0]
	fieldName := params[1]
	fieldValue := fl.Field().String()

	var count int64
	err := v.DB.Table(tableName).Where(fmt.Sprintf("%s = ?", fieldName), fieldValue).Count(&count).Error
	if err != nil {
		return false
	}

	return count == 0
}

// incolumnValidator
func (v *GoValidator) incolumnValidator(fl validator.FieldLevel) bool {
	params := strings.Split(fl.Param(), "->")
	if len(params) != 2 {
		return false
	}

	tableName := params[0]
	fieldName := params[1]
	fieldValue := fl.Field().String()

	if fieldValue == "" {
		return true
	}

	var count int64
	err := v.DB.Table(tableName).Where(fmt.Sprintf("%s = ?", fieldName), fieldValue).Count(&count).Error
	if err != nil {
		return false
	}

	return count != 0
}

// complexPasswordValidator
func (v *GoValidator) complexPasswordValidator(fl validator.FieldLevel) bool {
	password := fl.Field().String()
	if len(password) < 8 || len(password) > 64 {
		return false
	}
	var hasLower, hasUpper, hasNumber, hasSpecial bool
	specialChars := `!@#$%^&*()_-+={[}]| \:;"'<,>.?/~`

	for _, r := range password {
		switch {
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsDigit(r):
			hasNumber = true
		case strings.ContainsRune(specialChars, r):
			hasSpecial = true
		}

		if hasLower && hasUpper && hasNumber && hasSpecial {
			return true
		}
	}
	return hasLower && hasUpper && hasNumber && hasSpecial
}

// whiteSpaceValidator
func (v *GoValidator) whiteSpaceValidator(fl validator.FieldLevel) bool {
	text := fl.Field().String()
	return strings.IndexFunc(text, unicode.IsSpace) == -1
}

// username validator
func (v *GoValidator) usernameValidator(fl validator.FieldLevel) bool {
	username := fl.Field().String()

	return usernameRegex.MatchString(username)
}

// Validate implements the Validator interface, validating the given data and returning a ValidationError if there are any validation errors
func (v *GoValidator) Validate(ctx context.Context, data interface{}) error {
	err := v.validate.StructCtx(ctx, data)
	if err == nil {
		return nil
	}

	if _, ok := err.(*validator.InvalidValidationError); ok {
		return err
	}

	validationErrors := err.(validator.ValidationErrors)
	if len(validationErrors) > 0 {
		errorFields := make(map[string]string)
		isConflict := false
		isNotFound := false
		hasFormatError := false

		for _, err := range validationErrors {
			switch err.Tag() {
			case "unique":
				isConflict = true
			case "incolumn":
				isNotFound = true
			default:
				hasFormatError = true
			}
			errorFields[err.Field()] = err.Translate(v.uni)
		}

		var errMsgs []string
		for field, msg := range errorFields {
			errMsgs = append(errMsgs, fmt.Sprintf("%s: %s", field, msg))
		}
		detail := strings.Join(errMsgs, "; ")

		// 422
		if hasFormatError {
			return customerrors.NewValidationError(detail)
		}
		// 404
		if isNotFound {
			return customerrors.NewNotFoundError(detail)
		}
		// 409
		if isConflict {
			return customerrors.NewConflictError(detail)
		}

		return customerrors.NewValidationError(detail)
	}
	return nil
}

func (ve *ValidationError) Error() string {
	var errMsgs []string
	for field, msg := range ve.ErrorFields {
		errMsgs = append(errMsgs, fmt.Sprintf("%s: %s", field, msg))
	}
	return strings.Join(errMsgs, "; ")
}

// Helper function to convert snake_case to Proper Case (for better error messages)
func toProperCase(input string) string {
	// snake case
	input = strings.ReplaceAll(input, "_", " ")

	// camel case
	var spacedString strings.Builder
	for i, r := range input {
		if i > 0 && unicode.IsUpper(r) && input[i-1] != ' ' {
			spacedString.WriteRune(' ')
		}
		spacedString.WriteRune(r)
	}

	// make first word capital
	words := strings.Fields(spacedString.String())
	for i, word := range words {
		runes := []rune(word)
		runes[0] = unicode.ToUpper(runes[0])

		for j := 1; j < len(runes); j++ {
			runes[j] = unicode.ToLower(runes[j])
		}
		words[i] = string(runes)
	}

	return strings.Join(words, " ")
}
