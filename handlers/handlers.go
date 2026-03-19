package handlers

import (
	"encoding/json"
	"net/http"

	"smtp-gateway/mailer"
	"smtp-gateway/models"
	"smtp-gateway/validation"
)

// SendEmail handles POST /api/send
func SendEmail(w http.ResponseWriter, r *http.Request) {
	var req models.SendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body", nil)
		return
	}

	if errs := validation.Validate(&req); errs != nil {
		writeError(w, http.StatusUnprocessableEntity, "validation failed", errs)
		return
	}

	if err := mailer.Send(&req); err != nil {
		writeError(w, http.StatusBadGateway, err.Error(), nil)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(models.SendResponse{
		Success: true,
		Message: "email sent successfully",
	})
}

// Health handles GET /api/health
func Health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// writeError writes a structured JSON error response.
func writeError(w http.ResponseWriter, code int, msg string, details map[string]string) {
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(models.ErrorResponse{
		Error:   msg,
		Details: details,
	})
}
