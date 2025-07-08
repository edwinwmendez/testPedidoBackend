package repositories

import (
	"backend/internal/models"

	"gorm.io/gorm"
)

type UserRepository interface {
	Create(user *models.User) error
	FindByID(id string) (*models.User, error)
	FindByEmail(email string) (*models.User, error)
	FindByRole(role models.UserRole) ([]*models.User, error)
	FindAll() ([]*models.User, error)
	FindAllWithPagination(offset, limit int, role *models.UserRole) ([]*models.User, int64, error)
	Update(user *models.User) error
	Delete(id string) error
	SoftDelete(id string) error
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{
		db: db,
	}
}

func (r *userRepository) Create(user *models.User) error {
	return r.db.Create(user).Error
}

func (r *userRepository) FindByID(id string) (*models.User, error) {
	var user models.User

	if err := r.db.Where("user_id = ?", id).First(&user).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *userRepository) FindByEmail(email string) (*models.User, error) {
	var user models.User

	if err := r.db.Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *userRepository) FindByRole(role models.UserRole) ([]*models.User, error) {
	var users []*models.User

	if err := r.db.Where("user_role = ?", role).Find(&users).Error; err != nil {
		return nil, err
	}

	return users, nil
}

func (r *userRepository) Update(user *models.User) error {
	return r.db.Save(user).Error
}

func (r *userRepository) FindAll() ([]*models.User, error) {
	var users []*models.User

	if err := r.db.Find(&users).Error; err != nil {
		return nil, err
	}

	return users, nil
}

func (r *userRepository) FindAllWithPagination(offset, limit int, role *models.UserRole) ([]*models.User, int64, error) {
	var users []*models.User
	var total int64

	query := r.db.Model(&models.User{})

	if role != nil {
		query = query.Where("user_role = ?", *role)
	}

	// Contar el total de registros
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Obtener los registros con paginación
	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

func (r *userRepository) Delete(id string) error {
	return r.db.Delete(&models.User{}, "user_id = ?", id).Error
}

func (r *userRepository) SoftDelete(id string) error {
	return r.db.Model(&models.User{}).Where("user_id = ?", id).Update("is_active", false).Error
}
