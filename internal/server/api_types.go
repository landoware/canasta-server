package server

type ErrorMessage struct {
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}
