package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/shiwano/errdef"
	"github.com/shiwano/errdef/examples/http_api/internal/errdefs"
	"github.com/shiwano/errdef/examples/http_api/internal/service"
)

// Handler handles HTTP requests for user operations.
type Handler struct {
	service service.Service
}

// New creates a new Handler instance.
func New(svc service.Service) *Handler {
	return &Handler{
		service: svc,
	}
}

// GetUser handles GET /users/:id requests.
func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := strings.TrimPrefix(r.URL.Path, "/users/")

	user, err := h.service.GetUser(ctx, id)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, http.StatusOK, user)
}

// CreateUser handles POST /users requests.
func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, r, errdefs.ErrValidation.With(ctx).New("invalid request body"))
		return
	}

	user, err := h.service.CreateUser(ctx, req.Name, req.Email)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, http.StatusCreated, user)
}

// UpdateUser handles PUT /users/:id requests.
func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := strings.TrimPrefix(r.URL.Path, "/users/")
	currentUserID := r.Header.Get("X-User-ID")

	if currentUserID == "" {
		h.writeError(w, r, errdefs.ErrUnauthorized.With(ctx).New("authentication required"))
		return
	}

	var req struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, r, errdefs.ErrValidation.With(ctx).New("invalid request body"))
		return
	}

	user, err := h.service.UpdateUser(ctx, id, req.Name, req.Email, currentUserID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, http.StatusOK, user)
}

// DeleteUser handles DELETE /users/:id requests.
func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := strings.TrimPrefix(r.URL.Path, "/users/")
	currentUserID := r.Header.Get("X-User-ID")

	if currentUserID == "" {
		h.writeError(w, r, errdefs.ErrUnauthorized.With(ctx).New("authentication required"))
		return
	}

	if err := h.service.DeleteUser(ctx, id, currentUserID); err != nil {
		h.writeError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// writeJSON writes a JSON response with the given status code.
func (h *Handler) writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("failed to encode response", "error", err)
	}
}

// writeError converts an error to a JSON response and logs it.
func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	ctx := r.Context()

	// Extract structured error information
	statusCode := errdef.HTTPStatusFrom.OrDefault(err, http.StatusInternalServerError)
	traceID := errdef.TraceIDFrom.OrZero(err)

	// Determine the error message to return
	message := err.Error()
	var errdefErr errdef.Error
	if errors.As(err, &errdefErr) {
		// If the error is not marked as public, use a generic message
		if !errdef.IsPublic(err) {
			message = "an internal error occurred"
		}
	}

	// Build error response
	resp := map[string]any{
		"error": message,
	}

	// Add kind if available
	if errors.As(err, &errdefErr) {
		resp["kind"] = string(errdefErr.Kind())
	}

	// Add trace_id if available
	if traceID != "" {
		resp["trace_id"] = traceID
	}

	// Add validation errors if present
	if validationErrs, ok := errdefs.ValidationErrorsFrom(err); ok {
		resp["validation_errors"] = validationErrs
	}

	// Add retry_after if present
	if retryAfter, ok := errdef.RetryAfterFrom(err); ok {
		resp["retry_after"] = retryAfter.Seconds()
	}

	// Log the error with structured fields
	logLevel := errdef.LogLevelFrom.OrDefault(err, slog.LevelError)
	slog.Log(ctx, logLevel, "request failed",
		"error", err,
		"method", r.Method,
		"path", r.URL.Path,
		"status", statusCode,
	)

	// Write JSON response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("failed to encode error response", "error", err)
	}
}
