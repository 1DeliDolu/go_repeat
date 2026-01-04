package validation

import (
	"errors"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

type FieldErrors map[string]string

// Bind/validation hatasını field->message map'e çevirir.
// dst: bind edilen struct pointer'ı (tag okumak için)
func FromBindError(err error, dst any) FieldErrors {
	out := FieldErrors{}

	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		for _, fe := range ve {
			key := fieldKey(dst, fe.StructField())
			out[key] = messageForTag(fe.Tag(), fe.Param())
		}
		return out
	}

	// Diğer bind hataları (tip mismatch vs)
	out["_"] = "Form verileri geçersiz."
	return out
}

func fieldKey(dst any, structField string) string {
	// form tag'ını bul (form:"email")
	t := reflect.TypeOf(dst)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return strings.ToLower(structField)
	}

	f, ok := t.FieldByName(structField)
	if !ok {
		return strings.ToLower(structField)
	}
	tag := f.Tag.Get("form")
	if tag == "" {
		return strings.ToLower(structField)
	}
	// form:"email,omitempty" gibi durumlarda virgül sonrası at
	if i := strings.Index(tag, ","); i >= 0 {
		tag = tag[:i]
	}
	if tag == "" || tag == "-" {
		return strings.ToLower(structField)
	}
	return tag
}

func messageForTag(tag, param string) string {
	switch tag {
	case "required":
		return "Bu alan zorunludur."
	case "email":
		return "Geçerli bir e-posta giriniz."
	case "min":
		return "En az " + param + " karakter olmalıdır."
	case "max":
		return "En fazla " + param + " karakter olmalıdır."
	default:
		return "Geçersiz değer."
	}
}
