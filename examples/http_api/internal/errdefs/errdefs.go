package errdefs

import "github.com/shiwano/errdef"

// Domain errors represent business logic errors that can occur during normal operation.
// Each error is configured with an appropriate HTTP status code and additional options.
var (
	// ErrNotFound indicates that a requested resource was not found.
	ErrNotFound = errdef.Define("not_found", errdef.HTTPStatus(404), errdef.Public())

	// ErrValidation indicates that input validation failed.
	ErrValidation = errdef.Define("validation", errdef.HTTPStatus(400), errdef.Public())

	// ErrUnauthorized indicates that authentication is required.
	ErrUnauthorized = errdef.Define("unauthorized", errdef.HTTPStatus(401), errdef.Public())

	// ErrForbidden indicates that the user doesn't have permission to access the resource.
	ErrForbidden = errdef.Define("forbidden", errdef.HTTPStatus(403))

	// ErrConflict indicates that the request conflicts with the current state.
	ErrConflict = errdef.Define("conflict", errdef.HTTPStatus(409), errdef.Public())

	// ErrRateLimited indicates that the rate limit has been exceeded.
	ErrRateLimited = errdef.Define("rate_limited", errdef.HTTPStatus(429), errdef.Retryable())

	// ErrDatabase represents database-related errors.
	ErrDatabase = errdef.Define("database", errdef.HTTPStatus(500))

	// ErrInternal represents unexpected internal errors.
	ErrInternal = errdef.Define("internal", errdef.HTTPStatus(500))
)

// Custom field definitions provide type-safe field attachment and extraction.
// These fields can be attached to errors to provide structured context.
var (
	// UserID field stores user identifiers.
	UserID, UserIDFrom = errdef.DefineField[string]("user_id")

	// Email field stores email addresses with automatic redaction in logs and JSON.
	Email, EmailFrom = errdef.DefineField[errdef.Redacted[string]]("email")

	// TenantID field stores tenant identifiers for multi-tenant applications.
	TenantID, TenantIDFrom = errdef.DefineField[string]("tenant_id")

	// ResourceType field indicates what type of resource an error is related to.
	ResourceType, ResourceTypeFrom = errdef.DefineField[string]("resource_type")

	// ValidationErrors field stores field-level validation error messages.
	ValidationErrors, ValidationErrorsFrom = errdef.DefineField[map[string]string]("validation_errors")
)
