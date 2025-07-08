package handlers

import (
	"backend/internal/auth"
	"backend/internal/models"
	"backend/internal/services"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

// UserHandler maneja las peticiones HTTP relacionadas con usuarios
type UserHandler struct {
	userService *services.UserService
}

// NewUserHandler crea un nuevo handler de usuarios
func NewUserHandler(userService *services.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

// UpdateUserRequest estructura para actualizar un usuario
type UpdateUserRequest struct {
	FullName    string `json:"full_name,omitempty"`
	PhoneNumber string `json:"phone_number,omitempty"`
}

// CreateUserRequest estructura para crear un usuario (admin)
type CreateUserRequest struct {
	Email       string          `json:"email" validate:"required,email"`
	Password    string          `json:"password" validate:"required,min=6"`
	FullName    string          `json:"full_name" validate:"required"`
	PhoneNumber string          `json:"phone_number" validate:"required"`
	UserRole    models.UserRole `json:"user_role" validate:"required"`
}

// UpdateUserAdminRequest estructura para actualizar un usuario (admin)
type UpdateUserAdminRequest struct {
	Email       string          `json:"email,omitempty"`
	FullName    string          `json:"full_name,omitempty"`
	PhoneNumber string          `json:"phone_number,omitempty"`
	UserRole    models.UserRole `json:"user_role,omitempty"`
}

// @Summary Obtener perfil de usuario actual
// @Description Obtiene el perfil del usuario autenticado
// @Tags usuarios
// @Accept json
// @Produce json
// @Success 200 {object} models.User
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Security BearerAuth
// @Router /users/me [get]
// GetCurrentUser obtiene el perfil del usuario autenticado
func (h *UserHandler) GetCurrentUser(c *fiber.Ctx) error {
	// Obtener el usuario autenticado del contexto
	claims := c.Locals("user").(*auth.Claims)

	// Obtener el usuario de la base de datos
	user, err := h.userService.GetByID(claims.UserID.String())
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Usuario no encontrado",
		})
	}

	return c.JSON(user)
}

// @Summary Actualizar perfil de usuario
// @Description Actualiza el perfil del usuario autenticado
// @Tags usuarios
// @Accept json
// @Produce json
// @Param user body UpdateUserRequest true "Datos del usuario a actualizar"
// @Success 200 {object} models.User
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /users/me [put]
// UpdateCurrentUser actualiza el perfil del usuario autenticado
func (h *UserHandler) UpdateCurrentUser(c *fiber.Ctx) error {
	// Obtener el usuario autenticado del contexto
	claims := c.Locals("user").(*auth.Claims)

	// Parsear el cuerpo de la petición
	var req UpdateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Datos inválidos",
		})
	}

	// Obtener el usuario actual
	user, err := h.userService.GetByID(claims.UserID.String())
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Usuario no encontrado",
		})
	}

	// Actualizar los campos del usuario
	if req.FullName != "" {
		user.FullName = req.FullName
	}

	if req.PhoneNumber != "" {
		user.PhoneNumber = req.PhoneNumber
	}

	// Guardar los cambios
	if err := h.userService.Update(user); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error al actualizar el usuario",
		})
	}

	return c.JSON(user)
}

// @Summary Obtener todos los usuarios
// @Description Obtiene la lista de todos los usuarios (solo para administradores)
// @Tags usuarios
// @Accept json
// @Produce json
// @Param role query string false "Filtrar por rol (CLIENT, REPARTIDOR, ADMIN)"
// @Success 200 {array} models.User
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /users [get]
// GetAllUsers obtiene todos los usuarios (solo para administradores)
func (h *UserHandler) GetAllUsers(c *fiber.Ctx) error {
	// Obtener parámetros de consulta opcionales
	role := c.Query("role")

	var users []*models.User
	var err error

	// Filtrar por rol si se proporciona
	if role != "" {
		users, err = h.userService.GetByRole(models.UserRole(role))
	} else {
		users, err = h.userService.GetAll()
	}

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error al obtener los usuarios",
		})
	}

	return c.JSON(users)
}

// @Summary Obtener un usuario por ID
// @Description Obtiene los detalles de un usuario específico por su ID (solo para administradores)
// @Tags usuarios
// @Accept json
// @Produce json
// @Param id path string true "ID del usuario"
// @Success 200 {object} models.User
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Security BearerAuth
// @Router /users/{id} [get]
// GetUserByID obtiene un usuario por su ID (solo para administradores)
func (h *UserHandler) GetUserByID(c *fiber.Ctx) error {
	// Obtener el ID del usuario de los parámetros
	userID := c.Params("id")
	if userID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID de usuario requerido",
		})
	}

	// Obtener el usuario
	user, err := h.userService.GetByID(userID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Usuario no encontrado",
		})
	}

	return c.JSON(user)
}

// @Summary Listar todos los usuarios con paginación
// @Description Obtiene la lista de usuarios con paginación y filtros (solo para administradores)
// @Tags admin
// @Accept json
// @Produce json
// @Param page query int false "Página (por defecto: 1)"
// @Param page_size query int false "Tamaño de página (por defecto: 10, máximo: 100)"
// @Param role query string false "Filtrar por rol (CLIENT, REPARTIDOR, ADMIN)"
// @Success 200 {object} services.PaginatedUsersResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /admin/users [get]
// GetAllUsersAdmin obtiene todos los usuarios con paginación (solo para administradores)
func (h *UserHandler) GetAllUsersAdmin(c *fiber.Ctx) error {
	// Obtener parámetros de paginación
	page, _ := strconv.Atoi(c.Query("page", "1"))
	pageSize, _ := strconv.Atoi(c.Query("page_size", "10"))
	roleFilter := c.Query("role")

	// Validar rol si se proporciona
	if roleFilter != "" {
		switch models.UserRole(roleFilter) {
		case models.UserRoleClient, models.UserRoleRepartidor, models.UserRoleAdmin:
			// Rol válido
		default:
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Rol inválido. Los roles válidos son: CLIENT, REPARTIDOR, ADMIN",
			})
		}
	}

	// Obtener usuarios con paginación
	result, err := h.userService.GetUsersWithPagination(page, pageSize, roleFilter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error al obtener los usuarios",
		})
	}

	return c.JSON(result)
}

// @Summary Crear un nuevo usuario
// @Description Crea un nuevo usuario en el sistema (solo para administradores)
// @Tags admin
// @Accept json
// @Produce json
// @Param user body CreateUserRequest true "Datos del usuario a crear"
// @Success 201 {object} models.User
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 409 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /admin/users [post]
// CreateUserAdmin crea un nuevo usuario (solo para administradores)
func (h *UserHandler) CreateUserAdmin(c *fiber.Ctx) error {
	// Obtener el admin del contexto
	claims := c.Locals("user").(*auth.Claims)

	// Parsear el cuerpo de la petición
	var req CreateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Datos inválidos",
		})
	}

	// Validar campos requeridos
	if req.Email == "" || req.Password == "" || req.FullName == "" || req.PhoneNumber == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Todos los campos son requeridos: email, password, full_name, phone_number, user_role",
		})
	}

	// Validar rol
	switch req.UserRole {
	case models.UserRoleClient, models.UserRoleRepartidor, models.UserRoleAdmin:
		// Rol válido
	default:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Rol inválido. Los roles válidos son: CLIENT, REPARTIDOR, ADMIN",
		})
	}

	// Crear el usuario
	user := &models.User{
		Email:       req.Email,
		FullName:    req.FullName,
		PhoneNumber: req.PhoneNumber,
		UserRole:    req.UserRole,
		IsActive:    true,
	}

	// Establecer la contraseña
	if err := user.SetPassword(req.Password); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error al procesar la contraseña",
		})
	}

	// Crear el usuario usando el servicio
	if err := h.userService.CreateUserAdmin(user, claims.UserID.String()); err != nil {
		if err == services.ErrEmailAlreadyExists {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "El correo electrónico ya está en uso",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error al crear el usuario",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(user)
}

// @Summary Actualizar un usuario
// @Description Actualiza los datos de un usuario existente (solo para administradores)
// @Tags admin
// @Accept json
// @Produce json
// @Param id path string true "ID del usuario"
// @Param user body UpdateUserAdminRequest true "Datos del usuario a actualizar"
// @Success 200 {object} models.User
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /admin/users/{id} [put]
// UpdateUserAdmin actualiza un usuario existente (solo para administradores)
func (h *UserHandler) UpdateUserAdmin(c *fiber.Ctx) error {
	// Obtener el admin del contexto
	claims := c.Locals("user").(*auth.Claims)

	// Obtener el ID del usuario de los parámetros
	userID := c.Params("id")
	if userID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID de usuario requerido",
		})
	}

	// Parsear el cuerpo de la petición
	var req UpdateUserAdminRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Datos inválidos",
		})
	}

	// Obtener el usuario actual
	user, err := h.userService.GetByID(userID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Usuario no encontrado",
		})
	}

	// Actualizar los campos del usuario
	if req.Email != "" {
		user.Email = req.Email
	}
	if req.FullName != "" {
		user.FullName = req.FullName
	}
	if req.PhoneNumber != "" {
		user.PhoneNumber = req.PhoneNumber
	}
	if req.UserRole != "" {
		// Validar rol
		switch req.UserRole {
		case models.UserRoleClient, models.UserRoleRepartidor, models.UserRoleAdmin:
			user.UserRole = req.UserRole
		default:
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Rol inválido. Los roles válidos son: CLIENT, REPARTIDOR, ADMIN",
			})
		}
	}

	// Actualizar el usuario usando el servicio
	if err := h.userService.UpdateUserAdmin(user, claims.UserID.String()); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error al actualizar el usuario",
		})
	}

	return c.JSON(user)
}

// @Summary Activar un usuario
// @Description Activa un usuario inactivo (solo para administradores)
// @Tags admin
// @Accept json
// @Produce json
// @Param id path string true "ID del usuario"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /admin/users/{id}/activate [put]
// ActivateUserAdmin activa un usuario (solo para administradores)
func (h *UserHandler) ActivateUserAdmin(c *fiber.Ctx) error {
	// Obtener el admin del contexto
	claims := c.Locals("user").(*auth.Claims)

	// Obtener el ID del usuario de los parámetros
	userID := c.Params("id")
	if userID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID de usuario requerido",
		})
	}

	// Activar el usuario usando el servicio
	if err := h.userService.ActivateUser(userID, claims.UserID.String()); err != nil {
		if err == services.ErrUserNotFoundService {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Usuario no encontrado",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error al activar el usuario",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Usuario activado exitosamente",
	})
}

// @Summary Desactivar un usuario
// @Description Desactiva un usuario activo (solo para administradores)
// @Tags admin
// @Accept json
// @Produce json
// @Param id path string true "ID del usuario"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /admin/users/{id}/deactivate [put]
// DeactivateUserAdmin desactiva un usuario (solo para administradores)
func (h *UserHandler) DeactivateUserAdmin(c *fiber.Ctx) error {
	// Obtener el admin del contexto
	claims := c.Locals("user").(*auth.Claims)

	// Obtener el ID del usuario de los parámetros
	userID := c.Params("id")
	if userID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID de usuario requerido",
		})
	}

	// Desactivar el usuario usando el servicio
	if err := h.userService.DeactivateUser(userID, claims.UserID.String()); err != nil {
		if err == services.ErrUserNotFoundService {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Usuario no encontrado",
			})
		}
		if err == services.ErrCannotDeactivateAdmin {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "No se puede desactivar un usuario administrador por motivos de seguridad",
			})
		}
		if err == services.ErrCannotDeactivateSelf {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "No puedes desactivarte a ti mismo",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error al desactivar el usuario",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Usuario desactivado exitosamente",
	})
}

// @Summary Eliminar un usuario
// @Description Elimina permanentemente un usuario del sistema (solo para administradores)
// @Tags admin
// @Accept json
// @Produce json
// @Param id path string true "ID del usuario"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /admin/users/{id} [delete]
// DeleteUserAdmin elimina un usuario (solo para administradores)
func (h *UserHandler) DeleteUserAdmin(c *fiber.Ctx) error {
	// Obtener el admin del contexto
	claims := c.Locals("user").(*auth.Claims)

	// Obtener el ID del usuario de los parámetros
	userID := c.Params("id")
	if userID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID de usuario requerido",
		})
	}

	// Eliminar el usuario usando el servicio
	if err := h.userService.DeleteUserAdmin(userID, claims.UserID.String()); err != nil {
		if err == services.ErrUserNotFoundService {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Usuario no encontrado",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error al eliminar el usuario",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Usuario eliminado exitosamente",
	})
}

// RegisterRoutes registra las rutas del handler en el router
func (h *UserHandler) RegisterRoutes(router fiber.Router, authMiddleware fiber.Handler, adminOnly fiber.Handler) {
	users := router.Group("/users", authMiddleware)

	// Rutas para todos los usuarios autenticados
	users.Get("/me", h.GetCurrentUser)
	users.Put("/me", h.UpdateCurrentUser)

	// Rutas solo para administradores
	users.Get("/", adminOnly, h.GetAllUsers)
	users.Get("/:id", adminOnly, h.GetUserByID)

	// Rutas de administración de usuarios
	admin := router.Group("/admin", authMiddleware, adminOnly)
	adminUsers := admin.Group("/users")

	// Endpoints de administración de usuarios
	adminUsers.Get("/", h.GetAllUsersAdmin)                  // GET /admin/users
	adminUsers.Post("/", h.CreateUserAdmin)                  // POST /admin/users
	adminUsers.Put("/:id", h.UpdateUserAdmin)                // PUT /admin/users/{id}
	adminUsers.Put("/:id/activate", h.ActivateUserAdmin)     // PUT /admin/users/{id}/activate
	adminUsers.Put("/:id/deactivate", h.DeactivateUserAdmin) // PUT /admin/users/{id}/deactivate
	adminUsers.Delete("/:id", h.DeleteUserAdmin)             // DELETE /admin/users/{id}
}
