package services

import "github.com/google/uuid"

// mustParseUUID parsea un string a UUID, panic si falla (para uso interno)
func mustParseUUID(s string) uuid.UUID {
	u, err := uuid.Parse(s)
	if err != nil {
		panic("invalid UUID: " + s)
	}
	return u
}