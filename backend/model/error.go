package model

type ErrorKind string

const (
	ErrorInvalidInput ErrorKind = "invalid_input"
	ErrorUnauthorized ErrorKind = "unauthorized"
	ErrorForbidden    ErrorKind = "forbidden"
	ErrorNotFound     ErrorKind = "not_found"
	ErrorConflict     ErrorKind = "conflict"
	ErrorInternal     ErrorKind = "internal"
)

type ErrorDetail struct {
	Kind   ErrorKind `json:"kind"`
	Detail string    `json:"detail"`
}

func (e *ErrorDetail) Error() string {
	return e.Detail
}

func InvalidInput(detail string) *ErrorDetail {
	return &ErrorDetail{Kind: ErrorInvalidInput, Detail: detail}
}

func Unauthorized(detail string) *ErrorDetail {
	return &ErrorDetail{Kind: ErrorUnauthorized, Detail: detail}
}

func Forbidden(detail string) *ErrorDetail {
	return &ErrorDetail{Kind: ErrorForbidden, Detail: detail}
}

func NotFound(detail string) *ErrorDetail {
	return &ErrorDetail{Kind: ErrorNotFound, Detail: detail}
}

func Conflict(detail string) *ErrorDetail {
	return &ErrorDetail{Kind: ErrorConflict, Detail: detail}
}

func InternalError(detail string) *ErrorDetail {
	return &ErrorDetail{Kind: ErrorInternal, Detail: detail}
}
