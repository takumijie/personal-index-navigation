package service

import "net/http"

type Error struct {
	Status  int
	Code    string
	Message string
}

func (e *Error) Error() string { return e.Message }

func failure(status int, code, message string) *Error {
	return &Error{Status: status, Code: code, Message: message}
}

func invalid(message string) *Error {
	return failure(http.StatusBadRequest, "invalid_request", message)
}

func notFound(message string) *Error {
	return failure(http.StatusNotFound, "not_found", message)
}
