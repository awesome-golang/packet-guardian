package common

import (
	"encoding/json"
	"net/http"
)

// A APIResponse is returned as a JSON struct to the client
type APIResponse struct {
	Message string
	Data    interface{}
}

// NewAPIResponse creates an APIResponse object with status c, message m, and data d
func NewAPIResponse(m string, d interface{}) *APIResponse {
	return &APIResponse{
		Message: m,
		Data:    d,
	}
}

// Encode the APIResponse into JSON
func (a *APIResponse) Encode() []byte {
	b, err := json.Marshal(a)
	if err != nil {
		// Do something
	}
	return b
}

func (a *APIResponse) WriteResponse(w http.ResponseWriter, code int) (int64, error) {
	r := a.Encode()
	l := len(r)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	if code == http.StatusNoContent {
		return 0, nil
	}
	w.Write(r)
	return int64(l), nil
}
