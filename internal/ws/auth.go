package ws

import (
	"errors"
	"fmt"

	"backend/config"
	"backend/internal/auth"
	"backend/internal/models"

	"github.com/golang-jwt/jwt/v5"
)

// ValidateWebSocketToken valida el JWT recibido y retorna el userID y rol.
func ValidateWebSocketToken(tokenString string, cfg *config.Config) (userID string, role string, err error) {
	if tokenString == "" {
		return "", "", errors.New("token vacío")
	}

	// Parsear el token usando la misma lógica que el servicio de auth
	token, err := jwt.ParseWithClaims(tokenString, &auth.Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Verificar el método de firma
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("método de firma inesperado: %v", token.Header["alg"])
		}
		return []byte(cfg.JWT.Secret), nil
	})

	if err != nil {
		return "", "", fmt.Errorf("error al parsear token: %w", err)
	}

	if !token.Valid {
		return "", "", errors.New("token inválido")
	}

	// Extraer los claims
	claims, ok := token.Claims.(*auth.Claims)
	if !ok {
		return "", "", errors.New("claims inválidos")
	}

	// Convertir el rol a string para WebSocket
	var roleStr string
	switch claims.UserRole {
	case models.UserRoleAdmin:
		roleStr = "ADMIN"
	case models.UserRoleRepartidor:
		roleStr = "REPARTIDOR"
	case models.UserRoleClient:
		roleStr = "CLIENT"
	default:
		return "", "", errors.New("rol de usuario inválido")
	}

	return claims.UserID.String(), roleStr, nil
}
