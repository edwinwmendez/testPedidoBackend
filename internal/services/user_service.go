package services

import (
	"errors"
	"log"

	"backend/internal/models"
	"backend/internal/repositories"
)

var (
	ErrUserNotFoundService = errors.New("usuario no encontrado")
	ErrEmailAlreadyExists  = errors.New("el correo electrónico ya está en uso")
	ErrPhoneAlreadyExists  = errors.New("el número de teléfono ya está en uso")
	ErrCannotDeactivateAdmin = errors.New("no se puede desactivar un usuario administrador")
	ErrCannotDeactivateSelf  = errors.New("no puedes desactivarte a ti mismo")
)

// UserService maneja la lógica de negocio relacionada con usuarios
type UserService struct {
	repo repositories.UserRepository
}

// NewUserService crea un nuevo servicio de usuarios
func NewUserService(repo repositories.UserRepository) *UserService {
	return &UserService{
		repo: repo,
	}
}

// Create crea un nuevo usuario
func (s *UserService) Create(user *models.User) error {
	return s.repo.Create(user)
}

// GetByID obtiene un usuario por su ID
func (s *UserService) GetByID(id string) (*models.User, error) {
	return s.repo.FindByID(id)
}

// GetByEmail obtiene un usuario por su correo electrónico
func (s *UserService) GetByEmail(email string) (*models.User, error) {
	return s.repo.FindByEmail(email)
}

// GetByRole obtiene usuarios por su rol
func (s *UserService) GetByRole(role models.UserRole) ([]*models.User, error) {
	return s.repo.FindByRole(role)
}

// GetAll obtiene todos los usuarios
func (s *UserService) GetAll() ([]*models.User, error) {
	return s.repo.FindAll()
}

// Update actualiza un usuario existente
func (s *UserService) Update(user *models.User) error {
	return s.repo.Update(user)
}

// Delete elimina un usuario por su ID
func (s *UserService) Delete(id string) error {
	return s.repo.Delete(id)
}

// PaginatedUsersResponse estructura para respuesta paginada
type PaginatedUsersResponse struct {
	Users      []*models.User `json:"users"`
	TotalCount int64          `json:"total_count"`
	Page       int            `json:"page"`
	PageSize   int            `json:"page_size"`
	TotalPages int            `json:"total_pages"`
}

// GetUsersWithPagination obtiene usuarios con paginación y filtros
func (s *UserService) GetUsersWithPagination(page, pageSize int, roleFilter string) (*PaginatedUsersResponse, error) {
	// Validar parámetros de paginación
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	offset := (page - 1) * pageSize

	var role *models.UserRole
	if roleFilter != "" {
		roleValue := models.UserRole(roleFilter)
		role = &roleValue
	}

	users, total, err := s.repo.FindAllWithPagination(offset, pageSize, role)
	if err != nil {
		return nil, err
	}

	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))

	return &PaginatedUsersResponse{
		Users:      users,
		TotalCount: total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// CreateUserAdmin crea un nuevo usuario (solo administradores)
func (s *UserService) CreateUserAdmin(user *models.User, adminID string) error {
	log.Printf("Admin %s creando nuevo usuario: %s (%s)", adminID, user.Email, user.UserRole)
	
	// Verificar si el email ya existe
	if existingUser, _ := s.repo.FindByEmail(user.Email); existingUser != nil {
		return ErrEmailAlreadyExists
	}

	err := s.repo.Create(user)
	if err != nil {
		log.Printf("Error al crear usuario %s: %v", user.Email, err)
		return err
	}

	log.Printf("Usuario %s creado exitosamente por admin %s", user.Email, adminID)
	return nil
}

// UpdateUserAdmin actualiza un usuario existente (solo administradores)
func (s *UserService) UpdateUserAdmin(user *models.User, adminID string) error {
	log.Printf("Admin %s actualizando usuario: %s", adminID, user.UserID.String())
	
	err := s.repo.Update(user)
	if err != nil {
		log.Printf("Error al actualizar usuario %s: %v", user.UserID.String(), err)
		return err
	}

	log.Printf("Usuario %s actualizado exitosamente por admin %s", user.UserID.String(), adminID)
	return nil
}

// ActivateUser activa un usuario
func (s *UserService) ActivateUser(userID string, adminID string) error {
	log.Printf("Admin %s activando usuario: %s", adminID, userID)
	
	user, err := s.repo.FindByID(userID)
	if err != nil {
		return ErrUserNotFoundService
	}

	user.IsActive = true
	err = s.repo.Update(user)
	if err != nil {
		log.Printf("Error al activar usuario %s: %v", userID, err)
		return err
	}

	log.Printf("Usuario %s activado exitosamente por admin %s", userID, adminID)
	return nil
}

// DeactivateUser desactiva un usuario con validaciones de seguridad
func (s *UserService) DeactivateUser(userID string, adminID string) error {
	log.Printf("Admin %s intentando desactivar usuario: %s", adminID, userID)
	
	// Obtener el usuario a desactivar
	user, err := s.repo.FindByID(userID)
	if err != nil {
		return ErrUserNotFoundService
	}

	// VALIDACIÓN 1: No se puede desactivar un administrador
	if user.UserRole == models.UserRoleAdmin {
		log.Printf("❌ Intento de desactivar administrador bloqueado: %s", userID)
		return ErrCannotDeactivateAdmin
	}

	// VALIDACIÓN 2: Un admin no puede desactivarse a sí mismo
	if userID == adminID {
		log.Printf("❌ Intento de autodesactivación bloqueado: %s", adminID)
		return ErrCannotDeactivateSelf
	}

	// Si pasa todas las validaciones, proceder con la desactivación
	user.IsActive = false
	err = s.repo.Update(user)
	if err != nil {
		log.Printf("Error al desactivar usuario %s: %v", userID, err)
		return err
	}

	log.Printf("✅ Usuario %s desactivado exitosamente por admin %s", userID, adminID)
	return nil
}

// DeleteUserAdmin elimina un usuario (solo administradores)
func (s *UserService) DeleteUserAdmin(userID string, adminID string) error {
	log.Printf("Admin %s eliminando usuario: %s", adminID, userID)
	
	// Verificar que el usuario existe
	_, err := s.repo.FindByID(userID)
	if err != nil {
		return ErrUserNotFoundService
	}

	err = s.repo.Delete(userID)
	if err != nil {
		log.Printf("Error al eliminar usuario %s: %v", userID, err)
		return err
	}

	log.Printf("Usuario %s eliminado exitosamente por admin %s", userID, adminID)
	return nil
}
