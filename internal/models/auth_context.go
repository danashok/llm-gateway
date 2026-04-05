package models

import "github.com/google/uuid"

type AuthContext struct {
	UserID      uuid.UUID
	TeamID      uuid.UUID
	DeveloperID uuid.UUID // Individual developer making the request
}
