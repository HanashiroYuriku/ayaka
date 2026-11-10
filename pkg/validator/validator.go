// Package validator wraps go-playground/validator with custom, database-aware
// rules and converts failures into apperror values.
package validator

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"unicode"

	"be-ayaka/internal/core/apperror"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	entranslations "github.com/go-playground/validator/v10/translations/en"
	"gorm.io/gorm"
)

// Validator validates request payloads.
type Validator interface {
	Validate(ctx context.Context, data any) error
}

var (
	usernameRegex   = regexp.MustCompile(`^[a-zA-Z0-9_.-]+$`)
	identifierRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
)

const specialChars = `!@#$%^&*()_-+={[}]|\:;"'<,>.?/~` + "`"

// GoValidator implements Validator using go-playground/validator.
type GoValidator struct {
	validate *validator.Validate
	trans    ut.Translator
	db       *gorm.DB
	// softDelete caches whether a table has a deleted_at column.
	softDelete sync.Map
}

// New creates a validator. db may be nil when the DB-aware tags
// (unique, incolumn) are not used, e.g. in unit tests.
func New(db *gorm.DB) *GoValidator {
	v := validator.New(validator.WithRequiredStructEnabled())
	enLocale := en.New()
	trans, _ := ut.New(enLocale, enLocale).GetTranslator("en")
	_ = entranslations.RegisterDefaultTranslations(v, trans)

	// Report JSON/query field names instead of Go struct field names.
	v.RegisterTagNameFunc(func(f reflect.StructField) string {
		for _, tag := range []string{"json", "query", "params", "form"} {
			name := strings.SplitN(f.Tag.Get(tag), ",", 2)[0]
			if name == "-" {
				return ""
			}
			if name != "" {
				return name
			}
		}
		return f.Name
	})

	gv := &GoValidator{validate: v, trans: trans, db: db}
	gv.registerRules()
	return gv
}

// Validate validates data and returns:
//   - apperror KindValidation (422) for format errors,
//   - apperror KindConflict (409) when only `unique` rules failed,
//   - apperror KindNotFound (404) when only `incolumn` rules failed.
func (v *GoValidator) Validate(ctx context.Context, data any) error {
	err := v.validate.StructCtx(ctx, data)
	if err == nil {
		return nil
	}

	var vErrs validator.ValidationErrors
	if !errors.As(err, &vErrs) {
		return apperror.Internal(fmt.Errorf("validate: %w", err))
	}

	fields := make(map[string]string, len(vErrs))
	var conflict, notFound, format bool
	for _, fe := range vErrs {
		switch fe.Tag() {
		case "unique":
			conflict = true
		case "incolumn":
			notFound = true
		default:
			format = true
		}
		if _, exists := fields[fe.Field()]; !exists {
			fields[fe.Field()] = fe.Translate(v.trans)
		}
	}

	switch {
	case format:
		return apperror.Validation(fields)
	case notFound:
		return apperror.ErrNotFound.WithFields(fields)
	case conflict:
		return apperror.ErrConflict.WithFields(fields)
	}
	return apperror.Validation(fields)
}

func (v *GoValidator) registerRules() {
	rules := []struct {
		tag     string
		fn      validator.FuncCtx
		message string
	}{
		{"unique", v.unique, "{0} already exists"},
		{"incolumn", v.inColumn, "{0} does not exist"},
		{"complexpassword", complexPassword, "{0} must be 8-64 characters and contain uppercase, lowercase, number and special character"},
		{"whitespace", noWhitespace, "{0} cannot contain spaces"},
		{"username", username, "{0} can only contain letters, numbers, underscores, hyphens and dots"},
	}
	for _, r := range rules {
		_ = v.validate.RegisterValidationCtx(r.tag, r.fn)
		v.registerMessage(r.tag, r.message)
	}

	// Friendlier overrides of default messages.
	v.registerMessage("required", "{0} is a required field")
	v.registerMessage("email", "{0} must be a valid email address")
}

func (v *GoValidator) registerMessage(tag, msg string) {
	_ = v.validate.RegisterTranslation(tag, v.trans,
		func(t ut.Translator) error { return t.Add(tag, msg, true) },
		func(t ut.Translator, fe validator.FieldError) string {
			s, _ := t.T(tag, toProperCase(fe.Field()), fe.Param())
			return s
		},
	)
}

// unique=table->column: the value must not exist yet (soft-deleted rows ignored).
func (v *GoValidator) unique(ctx context.Context, fl validator.FieldLevel) bool {
	count, ok := v.count(ctx, fl)
	return ok && count == 0
}

// incolumn=table->column: the value must exist (empty values are skipped; combine with required).
func (v *GoValidator) inColumn(ctx context.Context, fl validator.FieldLevel) bool {
	if fl.Field().String() == "" {
		return true
	}
	count, ok := v.count(ctx, fl)
	return ok && count > 0
}

func (v *GoValidator) count(ctx context.Context, fl validator.FieldLevel) (int64, bool) {
	if v.db == nil {
		panic("validator: unique/incolumn tags require a database connection")
	}
	table, column, found := strings.Cut(fl.Param(), "->")
	// Table/column come from struct tags, but validate them anyway to make SQL
	// injection impossible even if a tag is built dynamically.
	if !found || !identifierRegex.MatchString(table) || !identifierRegex.MatchString(column) {
		panic(fmt.Sprintf("validator: invalid table->column param %q", fl.Param()))
	}

	q := v.db.WithContext(ctx).Table(table).Where(column+" = ?", fl.Field().Interface())
	if v.hasSoftDelete(table) {
		q = q.Where("deleted_at IS NULL")
	}

	var count int64
	if err := q.Count(&count).Error; err != nil {
		return 0, false
	}
	return count, true
}

func (v *GoValidator) hasSoftDelete(table string) bool {
	if cached, ok := v.softDelete.Load(table); ok {
		return cached.(bool)
	}
	has := v.db.Migrator().HasColumn(table, "deleted_at")
	v.softDelete.Store(table, has)
	return has
}

func complexPassword(_ context.Context, fl validator.FieldLevel) bool {
	password := fl.Field().String()
	if n := len([]rune(password)); n < 8 || n > 64 {
		return false
	}
	var lower, upper, digit, special bool
	for _, r := range password {
		switch {
		case unicode.IsLower(r):
			lower = true
		case unicode.IsUpper(r):
			upper = true
		case unicode.IsDigit(r):
			digit = true
		case strings.ContainsRune(specialChars, r):
			special = true
		}
	}
	return lower && upper && digit && special
}

func noWhitespace(_ context.Context, fl validator.FieldLevel) bool {
	return strings.IndexFunc(fl.Field().String(), unicode.IsSpace) == -1
}

func username(_ context.Context, fl validator.FieldLevel) bool {
	return usernameRegex.MatchString(fl.Field().String())
}

// toProperCase converts "displayName" / "display_name" into "Display Name".
func toProperCase(input string) string {
	input = strings.ReplaceAll(input, "_", " ")

	var b strings.Builder
	runes := []rune(input)
	for i, r := range runes {
		if i > 0 && unicode.IsUpper(r) && runes[i-1] != ' ' {
			b.WriteRune(' ')
		}
		b.WriteRune(r)
	}

	words := strings.Fields(b.String())
	for i, w := range words {
		rs := []rune(strings.ToLower(w))
		rs[0] = unicode.ToUpper(rs[0])
		words[i] = string(rs)
	}
	return strings.Join(words, " ")
}
