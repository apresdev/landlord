package apperr

import (
	"errors"
	"net/http"
	"strings"

	"go.uber.org/zap"
)

func ToHTTPStatus(err error) int {
	if IsNotFound(err) {
		return http.StatusNotFound
	}
	if IsAlreadyExists(err) {
		return http.StatusBadRequest
	}
	if IsBadRequest(err) {
		return http.StatusBadRequest
	}
	if IsInternal(err) {
		return http.StatusInternalServerError
	}
	if IsUnauthorized(err) {
		return http.StatusUnauthorized
	}
	zap.L().Error("could not translate error to HTTP status", zap.Error(err))
	return http.StatusInternalServerError
}

type NotFound struct {
	error
}

func NewNotFound(params ...string) error {
	if len(params) == 0 {
		return NotFound{error: errors.New("not found")}
	} else {
		return NotFound{error: errors.New(strings.Join(params, " "))}
	}
}

func IsNotFound(err error) bool {
	return errors.As(err, &NotFound{})
}

type Internal struct {
	error
}

func IsInternal(err error) bool {
	return errors.As(err, &Internal{})
}

func NewInternal(params ...string) error {
	if len(params) == 0 {
		return Internal{error: errors.New("internal error")}
	} else {
		return Internal{error: errors.New(strings.Join(params, " "))}
	}
}

type BadRequest struct {
	error
}

func IsBadRequest(err error) bool {
	return errors.As(err, &BadRequest{})
}

func NewBadRequest(params ...string) error {
	if len(params) == 0 {
		return BadRequest{error: errors.New("bad request")}
	} else {
		return BadRequest{error: errors.New(strings.Join(params, " "))}
	}
}

type AlreadyExists struct {
	error
}

func IsAlreadyExists(err error) bool {
	return errors.As(err, &AlreadyExists{})
}

func NewAlreadyExists(params ...string) error {
	if len(params) == 0 {
		return AlreadyExists{error: errors.New("already exists")}
	} else {
		return AlreadyExists{error: errors.New(strings.Join(params, " "))}
	}
}

type Unauthorized struct {
	error
}

func IsUnauthorized(err error) bool {
	return errors.As(err, &Unauthorized{})
}

func NewUnauthorized(params ...string) error {
	if len(params) == 0 {
		return Unauthorized{error: errors.New("unauthorized")}
	} else {
		return Unauthorized{error: errors.New(strings.Join(params, " "))}
	}
}
