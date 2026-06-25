package dto

type ErrorResponse struct {
	Success string      `json:"success"`
	Message string      `json:"message"`
	Error   interface{} `json:"errors"`
}
