package services

import (
	"time"
	"github.com/google/uuid"
)

// mustParseUUID parsea un string a UUID, panic si falla (para uso interno)
func mustParseUUID(s string) uuid.UUID {
	u, err := uuid.Parse(s)
	if err != nil {
		panic("invalid UUID: " + s)
	}
	return u
}

// getCurrentTimeString devuelve la fecha y hora actual como string
func getCurrentTimeString() string {
	return time.Now().Format(time.RFC3339)
}
