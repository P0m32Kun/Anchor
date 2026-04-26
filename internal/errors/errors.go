package errors

import "fmt"

type ErrorCode string

const (
	ErrScopeDenied   ErrorCode = "SCOPE_DENIED"
	ErrToolNotFound  ErrorCode = "TOOL_NOT_FOUND"
	ErrToolTimeout   ErrorCode = "TOOL_TIMEOUT"
	ErrToolExecution ErrorCode = "TOOL_EXECUTION"
	ErrParse         ErrorCode = "PARSE_ERROR"
	ErrTruncation    ErrorCode = "TRUNCATION_WARNING"
	ErrWorkdir       ErrorCode = "WORKDIR_ERROR"
	ErrNotFound      ErrorCode = "NOT_FOUND"
	ErrBadRequest    ErrorCode = "BAD_REQUEST"
	ErrInternal      ErrorCode = "INTERNAL_ERROR"
)

type AppError struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
	Detail  string    `json:"detail,omitempty"`
}

func New(code ErrorCode, message string) *AppError {
	return &AppError{Code: code, Message: message}
}

func Newf(code ErrorCode, format string, args ...interface{}) *AppError {
	return &AppError{Code: code, Message: fmt.Sprintf(format, args...)}
}

func (e *AppError) Error() string { return e.Message }

func (e *AppError) WithDetail(detail string) *AppError {
	e.Detail = detail
	return e
}
