package balancer

import (
	"encoding/json"
	"log"
	"net/http"
)

type ResponseMessage struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type ResponseKey struct {
	Key string `json:"key"`
}

func ResponseError(w http.ResponseWriter, message string, code int) {
	w.WriteHeader(code)

	response := ResponseMessage{
		Code:    code,
		Message: message,
	}

	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		log.Println("Failed to send error response:", err)
	}
}

func Response(w http.ResponseWriter, body any) {
	err := json.NewEncoder(w).Encode(body)
	if err != nil {
		log.Println("Failed to send JSON response:", err)
	}
}
