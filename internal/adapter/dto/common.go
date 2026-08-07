package dto

// ErrorResponse is the shape every error path returns. Handlers emit this as a
// gin.H literal; the two marshal identically, and this type is what the OpenAPI
// annotations reference.
type ErrorResponse struct {
	Error string `json:"error"`
}

// MessageResponse acknowledges a successful mutation that has no body to return.
type MessageResponse struct {
	Message string `json:"message"`
}
