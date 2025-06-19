package middlewares

import (
	"strings"

	"backend/internal/auth"
	"backend/internal/models"

	"github.com/gofiber/fiber/v2"
)

// AuthMiddleware verifica que el token JWT sea válido
func AuthMiddleware(authService auth.Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Obtener el token del header Authorization
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Se requiere token de autenticación",
			})
		}

		// Verificar que el token tenga el formato correcto
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Formato de token inválido",
			})
		}

		// Validar el token
		claims, err := authService.ValidateToken(parts[1])
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Token inválido o expirado",
			})
		}

		// Verificar si el usuario aún está activo
		user, err := authService.GetUserByID(claims.UserID)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Usuario no encontrado",
			})
		}

		if !user.IsActive {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Tu cuenta está inactiva. Contacta al administrador para reactivarla",
			})
		}

		// Almacenar los claims en el contexto para uso posterior
		c.Locals("user", claims)

		return c.Next()
	}
}

// RequireRole verifica que el usuario tenga al menos uno de los roles especificados
func RequireRole(roles ...models.UserRole) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Obtener los claims del contexto
		claims, ok := c.Locals("user").(*auth.Claims)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "No se encontró información de autenticación",
			})
		}

		// Verificar si el usuario tiene alguno de los roles requeridos
		for _, role := range roles {
			if claims.UserRole == role {
				return c.Next()
			}
		}

		// Si el usuario no tiene ninguno de los roles requeridos
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "No tienes permiso para acceder a este recurso",
		})
	}
}
