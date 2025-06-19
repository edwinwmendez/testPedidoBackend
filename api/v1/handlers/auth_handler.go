package handlers

import (
	"errors"
	"strings"

	"backend/internal/auth"
	"backend/internal/models"

	"github.com/gofiber/fiber/v2"
)

// AuthHandler maneja las rutas relacionadas con la autenticación
type AuthHandler struct {
	authService auth.Service
}

// NewAuthHandler crea una nueva instancia del manejador de autenticación
func NewAuthHandler(authService auth.Service) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// RegisterUserRequest representa los datos para registrar un nuevo usuario
type RegisterUserRequest struct {
	Email       string `json:"email" validate:"required,email"`
	Password    string `json:"password" validate:"required,min=6"`
	FullName    string `json:"full_name" validate:"required"`
	PhoneNumber string `json:"phone_number" validate:"required"`
	UserRole    string `json:"user_role" validate:"required,oneof=CLIENT REPARTIDOR ADMIN"`
}

// LoginRequest representa los datos para iniciar sesión
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// RefreshTokenRequest representa los datos para refrescar un token
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// RegisterResponse es la respuesta al registrar un usuario
type RegisterResponse struct {
	Message string `json:"message"`
	UserID  string `json:"user_id"`
}

// ErrorResponse es la respuesta de error estándar
type ErrorResponse struct {
	Error string `json:"error"`
}

// RegisterUser maneja la petición de registro de un nuevo usuario
// @Summary Registrar un nuevo usuario
// @Description Registra un nuevo usuario en el sistema
// @Tags autenticación
// @Accept json
// @Produce json
// @Param user body RegisterUserRequest true "Datos del usuario"
// @Success 201 {object} RegisterResponse
// @Failure 400 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /auth/register [post]
func (h *AuthHandler) RegisterUser(c *fiber.Ctx) error {
	var req RegisterUserRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Error al procesar la solicitud",
		})
	}

	// Validar campos
	if req.Email == "" || req.Password == "" || req.FullName == "" || req.PhoneNumber == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Todos los campos son requeridos",
		})
	}

	// Convertir string a UserRole
	var role models.UserRole
	switch strings.ToUpper(req.UserRole) {
	case "CLIENT":
		role = models.UserRoleClient
	case "REPARTIDOR":
		role = models.UserRoleRepartidor
	case "ADMIN":
		role = models.UserRoleAdmin
	default:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Rol de usuario inválido",
		})
	}

	// Registrar usuario
	user, err := h.authService.RegisterUser(req.Email, req.Password, req.FullName, req.PhoneNumber, role)
	if err != nil {
		if errors.Is(err, auth.ErrUserAlreadyExists) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "El email o teléfono ya está registrado",
			})
		}
		if errors.Is(err, auth.ErrInvalidRole) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Rol de usuario inválido",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error al registrar usuario",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Usuario registrado exitosamente",
		"user_id": user.UserID,
	})
}

// Login maneja la petición de inicio de sesión
// @Summary Iniciar sesión
// @Description Autentica un usuario y devuelve un token JWT
// @Tags autenticación
// @Accept json
// @Produce json
// @Param credentials body LoginRequest true "Credenciales de usuario"
// @Success 200 {object} auth.TokenPair
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /auth/login [post]
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Error al procesar la solicitud",
		})
	}

	// Validar campos
	if req.Email == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Email y contraseña son requeridos",
		})
	}

	// Autenticar usuario
	tokenPair, err := h.authService.Login(req.Email, req.Password)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Email o contraseña incorrectos",
			})
		}
		if errors.Is(err, auth.ErrUserInactive) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Tu cuenta está inactiva. Contacta al administrador para reactivarla",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error al iniciar sesión",
		})
	}

	return c.Status(fiber.StatusOK).JSON(tokenPair)
}

// RefreshToken maneja la petición de refrescar un token
// @Summary Refrescar token
// @Description Refresca un token JWT usando un token de refresco
// @Tags autenticación
// @Accept json
// @Produce json
// @Param refresh_token body RefreshTokenRequest true "Token de refresco"
// @Success 200 {object} auth.TokenPair
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *fiber.Ctx) error {
	var req RefreshTokenRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Error al procesar la solicitud",
		})
	}

	// Validar campos
	if req.RefreshToken == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Token de refresco es requerido",
		})
	}

	// Refrescar token
	tokenPair, err := h.authService.RefreshToken(req.RefreshToken)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidToken) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Token inválido o expirado",
			})
		}
		if errors.Is(err, auth.ErrUserInactive) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Tu cuenta está inactiva. Contacta al administrador para reactivarla",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error al refrescar token",
		})
	}

	return c.Status(fiber.StatusOK).JSON(tokenPair)
}

// Logout maneja el cierre de sesión
// @Summary Cerrar sesión
// @Description Cierra la sesión del usuario (en JWT stateless se maneja del lado del cliente)
// @Tags autenticación
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Security BearerAuth
// @Router /auth/logout [post]
func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	// En un sistema JWT stateless, el logout se maneja típicamente del lado del cliente
	// eliminando el token. Este endpoint es principalmente para cumplir con la API RESTful
	// y proporcionar una respuesta consistente.
	
	// En el futuro, aquí se podría implementar una blacklist de tokens
	// o invalidación en Redis si se requiere logout del lado del servidor
	
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Sesión cerrada exitosamente",
	})
}

// RegisterRoutes registra las rutas del manejador de autenticación
func (h *AuthHandler) RegisterRoutes(router fiber.Router) {
	authGroup := router.Group("/auth")

	authGroup.Post("/register", h.RegisterUser)
	authGroup.Post("/login", h.Login)
	authGroup.Post("/refresh", h.RefreshToken)
	authGroup.Post("/logout", h.Logout)
}
