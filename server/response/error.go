package response

import (
	"errors"
	"net/http"
)

type Error struct {
	Error  error `json:"error"`
	Status int   `json:"status"`
}

func NewError(err error, status ...int) *Error {
	if len(status) == 0 {
		status = []int{http.StatusInternalServerError}
	}

	return &Error{
		Error:  err,
		Status: status[0],
	}

}

func NewErrorFromString(message string, status ...int) *Error {
	return NewError(errors.New(message), status...)
}

func NewErrorFromStatus(status int) *Error {
	return NewErrorFromString(http.StatusText(status), status)
}
