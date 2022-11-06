package errorx

type SimpleMsg struct {
	Msg string `json:"msg"`
}

type ApiError struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func (e *ApiError) Error() string {
	return e.Msg
}

func NewApiError(code int, msg string) error {
	return &ApiError{Code: code, Msg: msg}
}

func NewApiErrorWithoutMsg(code int) error {
	return &ApiError{Code: code, Msg: ""}
}

func NewApiBadRequestError(msg string) error {
	return NewApiError(400, msg)
}

func NewApiNotFoundError(msg string) error {
	return NewApiError(404, msg)
}

func NewApiInternalServerError(msg string) error {
	return NewApiError(500, msg)
}
