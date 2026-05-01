package auth

import (
	"encoding/json"
	"errors"
	"net/http"

	appErrors "github.com/radamesvaz/bakery-app/internal/errors"
)

// apiErrorBody is the standard error envelope for tenant auth email flows (and shared auth handlers).
type apiErrorBody struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

func writeJSONAPIError(w http.ResponseWriter, status int, code, message string) {
	if message == "" {
		message = code
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(apiErrorBody{Error: code, Message: message})
}

func writeValidatorJSONError(w http.ResponseWriter, err error, fallback string) {
	var httpErr *appErrors.HTTPError
	if errors.As(err, &httpErr) {
		msg := httpErr.Error()
		code := "bad_request"
		if errors.Is(httpErr.Err, appErrors.ErrWeakPassword) {
			code = "weak_password"
		}
		writeJSONAPIError(w, httpErr.StatusCode, code, msg)
		return
	}
	writeJSONAPIError(w, http.StatusBadRequest, "bad_request", fallback)
}

func statusToDefaultCode(status int) string {
	switch status {
	case http.StatusBadRequest:
		return "bad_request"
	case http.StatusUnauthorized:
		return "unauthorized"
	case http.StatusForbidden:
		return "forbidden"
	case http.StatusNotFound:
		return "not_found"
	case http.StatusConflict:
		return "conflict"
	case http.StatusUnprocessableEntity:
		return "unprocessable_entity"
	case http.StatusTooManyRequests:
		return "too_many_requests"
	default:
		return "internal_error"
	}
}

func writeHTTPErrorAsJSON(w http.ResponseWriter, he *appErrors.HTTPError) {
	writeJSONAPIError(w, he.StatusCode, statusToDefaultCode(he.StatusCode), he.Error())
}

func writeTokenSemanticErrorJSON(w http.ResponseWriter, err error) {
	var code string
	switch {
	case errors.Is(err, appErrors.ErrInvalidToken):
		code = "invalid_token"
	case errors.Is(err, appErrors.ErrExpiredToken):
		code = "expired_token"
	case errors.Is(err, appErrors.ErrTokenAlreadyConsumed):
		code = "token_already_consumed"
	case errors.Is(err, appErrors.ErrTokenRevoked):
		code = "token_revoked"
	default:
		writeJSONAPIError(w, http.StatusUnprocessableEntity, "unprocessable_entity", err.Error())
		return
	}
	writeJSONAPIError(w, http.StatusUnprocessableEntity, code, err.Error())
}

// respondInvitationMutationError maps service errors for create/revoke/resend/accept.
// Returns true if the response was written (including HTTPError from services).
func respondInvitationMutationError(w http.ResponseWriter, err error) bool {
	switch {
	case errors.Is(err, appErrors.ErrForbidden):
		writeJSONAPIError(w, http.StatusForbidden, "forbidden", "Forbidden")
		return true
	case errors.Is(err, appErrors.ErrEmailAlreadyExists):
		writeJSONAPIError(w, http.StatusConflict, "email_already_exists", "Email already exists")
		return true
	case errors.Is(err, appErrors.ErrInvalidToken), errors.Is(err, appErrors.ErrExpiredToken),
		errors.Is(err, appErrors.ErrTokenAlreadyConsumed), errors.Is(err, appErrors.ErrTokenRevoked):
		writeTokenSemanticErrorJSON(w, err)
		return true
	}
	var he *appErrors.HTTPError
	if errors.As(err, &he) {
		writeHTTPErrorAsJSON(w, he)
		return true
	}
	return false
}

func respondForgotPasswordJSONError(w http.ResponseWriter, err error) {
	var he *appErrors.HTTPError
	if errors.As(err, &he) {
		writeHTTPErrorAsJSON(w, he)
		return
	}
	writeJSONAPIError(w, http.StatusInternalServerError, "internal_error", "Could not process request")
}

func respondPasswordResetError(w http.ResponseWriter, err error) bool {
	switch {
	case errors.Is(err, appErrors.ErrInvalidToken), errors.Is(err, appErrors.ErrExpiredToken),
		errors.Is(err, appErrors.ErrTokenAlreadyConsumed), errors.Is(err, appErrors.ErrTokenRevoked):
		writeTokenSemanticErrorJSON(w, err)
		return true
	}
	var he *appErrors.HTTPError
	if errors.As(err, &he) {
		writeHTTPErrorAsJSON(w, he)
		return true
	}
	return false
}
