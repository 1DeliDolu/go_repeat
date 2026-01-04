package apperr

import (
	"errors"
	"fmt"
	"net/http"
)


const (
	Invalid      Kind = "invalid"
	NotFound     Kind = "not_found"
	Unauthorized Kind = "unauthorized"
	Forbidden    Kind = "forbidden"
	Conflict     Kind = "conflict"
	Internal     Kind = "internal"
)



func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Kind, e.Err)
	}
	return string(e.Kind)
}

func (e *AppError) Unwrap() error { return e.Err }

// Constructors (PublicMsg kısa ve güvenli olmalı)
func InvalidErr(publicMsg string, fields map[string]string) *AppError {
	return &AppError{Kind: Invalid, PublicMsg: publicMsg, Fields: fields}
}
func NotFoundErr(publicMsg string) *AppError {
	return &AppError{Kind: NotFound, PublicMsg: publicMsg}
}
func UnauthorizedErr(publicMsg string) *AppError {
	return &AppError{Kind: Unauthorized, PublicMsg: publicMsg}
}
func ForbiddenErr(publicMsg string) *AppError {
	return &AppError{Kind: Forbidden, PublicMsg: publicMsg}
}
func ConflictErr(publicMsg string) *AppError {
	return &AppError{Kind: Conflict, PublicMsg: publicMsg}
}

// Wrap: internal hatayı public mesaj olmadan sar (default 500)
func Wrap(err error) *AppError {
	if err == nil {
		return nil
	}
	return &AppError{Kind: Internal, PublicMsg: "Beklenmeyen bir hata oluştu.", Err: err}
}

func As(err error) (*AppError, bool) {
	var ae *AppError
	if errors.As(err, &ae) {
		return ae, true
	}
	return nil, false
}

func HTTPStatus(err error) int {
	if ae, ok := As(err); ok {
		switch ae.Kind {
		case Invalid:
			return http.StatusBadRequest
		case Unauthorized:
			return http.StatusUnauthorized
		case Forbidden:
			return http.StatusForbidden
		case NotFound:
			return http.StatusNotFound
		case Conflict:
			return http.StatusConflict
		default:
			return http.StatusInternalServerError
		}
	}
	return http.StatusInternalServerError
}

func PublicMessage(err error) string {
	if ae, ok := As(err); ok && ae.PublicMsg != "" {
		return ae.PublicMsg
	}
	return "Beklenmeyen bir hata oluştu."
}
