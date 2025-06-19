package models

import (
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// UserRole define los roles posibles de un usuario
type UserRole string

const (
	UserRoleClient     UserRole = "CLIENT"
	UserRoleRepartidor UserRole = "REPARTIDOR"
	UserRoleAdmin      UserRole = "ADMIN"
)

// User representa un usuario en el sistema
type User struct {
	UserID       uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"user_id"`
	Email        string    `gorm:"type:varchar(255);not null;unique" json:"email"`
	PasswordHash string    `gorm:"type:varchar(255);not null" json:"-"` // No se envía en JSON
	FullName     string    `gorm:"type:varchar(255);not null" json:"full_name"`
	PhoneNumber  string    `gorm:"type:varchar(20);not null;unique" json:"phone_number"`
	UserRole     UserRole  `gorm:"type:varchar(20);not null" json:"user_role"`
	IsActive     bool      `gorm:"not null;default:true" json:"is_active"`
	CreatedAt    time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt    time.Time `gorm:"not null;default:now()" json:"updated_at"`
}

// BeforeCreate se ejecuta antes de crear un nuevo usuario
func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	// Si no se proporciona un ID, generamos uno
	if u.UserID == uuid.Nil {
		u.UserID = uuid.New()
	}
	return nil
}

// TableName especifica el nombre de la tabla para User
func (User) TableName() string {
	return "users"
}

// SetPassword establece la contraseña hasheada para el usuario
func (u *User) SetPassword(password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.PasswordHash = string(hashedPassword)
	return nil
}

// CheckPassword verifica si la contraseña proporcionada coincide con el hash almacenado
func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	return err == nil
}
