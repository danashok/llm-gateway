package errors

import "errors"

// Sentinel errors for the application
var (
	// ErrSessionNotFound indicates the session does not exist or the developer doesn't have access
	ErrSessionNotFound = errors.New("session not found")
)
