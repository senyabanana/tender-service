package models

// ErrorResponse описывает ошибку с кодом и сообщением.
type ErrorResponse struct {
	StatusCode int    `json:"-"`
	Message    string `json:"reason"`
}

// NewErrorResponse создает новую ошибку с кодом и сообщением.
func NewErrorResponse(statusCode int, message string) *ErrorResponse {
	return &ErrorResponse{
		StatusCode: statusCode,
		Message:    message}
}

// Реализация метода Error() для удовлетворения интерфейса error.
func (e *ErrorResponse) Error() string {
	return e.Message
}
