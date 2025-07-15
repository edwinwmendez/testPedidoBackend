package auth

import (
	"errors"
	"fmt"
	"log"
	"time"

	"backend/config"
	"backend/internal/models"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// Errores comunes
var (
	ErrInvalidCredentials = errors.New("credenciales inválidas")
	ErrUserAlreadyExists  = errors.New("el usuario ya existe")
	ErrInvalidToken       = errors.New("token inválido o expirado")
	ErrInvalidRole        = errors.New("rol de usuario inválido")
	ErrUserNotFound       = errors.New("usuario no encontrado")
	ErrUserInactive       = errors.New("usuario inactivo")
)

// Service interfaz para el servicio de autenticación
type Service interface {
	RegisterUser(email, password, fullName, phoneNumber string, role models.UserRole) (*models.User, error)
	Login(email, password string) (*TokenPair, error)
	ValidateToken(tokenString string) (*Claims, error)
	RefreshToken(refreshToken string) (*TokenPair, error)
	GetUserByID(id uuid.UUID) (*models.User, error)
}

// TokenPair representa un par de tokens (acceso y refresco)
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"` // Segundos hasta que el token de acceso expire
}

// Claims representa los datos a incluir en el token JWT
type Claims struct {
	UserID   uuid.UUID       `json:"user_id"`
	Email    string          `json:"email"`
	UserRole models.UserRole `json:"user_role"`
	jwt.RegisteredClaims
}

// service es la implementación del servicio de autenticación
type service struct {
	db     *gorm.DB
	config *config.Config
}

// NewService crea una nueva instancia del servicio de autenticación
func NewService(db *gorm.DB, cfg *config.Config) Service {
	return &service{
		db:     db,
		config: cfg,
	}
}

// RegisterUser registra un nuevo usuario en el sistema
func (s *service) RegisterUser(email, password, fullName, phoneNumber string, role models.UserRole) (*models.User, error) {
	// Verificar si el email ya existe
	var existingUser models.User
	if result := s.db.Where("email = ?", email).First(&existingUser); result.Error == nil {
		return nil, ErrUserAlreadyExists
	} else if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("error al verificar usuario existente: %w", result.Error)
	}

	// Verificar si el teléfono ya existe
	if result := s.db.Where("phone_number = ?", phoneNumber).First(&existingUser); result.Error == nil {
		return nil, ErrUserAlreadyExists
	} else if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("error al verificar teléfono existente: %w", result.Error)
	}

	// Validar el rol
	if role != models.UserRoleClient && role != models.UserRoleRepartidor && role != models.UserRoleAdmin {
		return nil, ErrInvalidRole
	}

	// Hashear la contraseña
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("error al hashear contraseña: %w", err)
	}

	// Crear el nuevo usuario
	user := models.User{
		Email:        email,
		PasswordHash: string(hashedPassword),
		FullName:     fullName,
		PhoneNumber:  phoneNumber,
		UserRole:     role,
	}

	// Guardar en la base de datos
	if err := s.db.Create(&user).Error; err != nil {
		return nil, fmt.Errorf("error al crear usuario: %w", err)
	}

	return &user, nil
}

// Login autentica a un usuario y retorna un par de tokens
func (s *service) Login(email, password string) (*TokenPair, error) {
	// Buscar al usuario por email
	var user models.User
	if err := s.db.Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("error al buscar usuario: %w", err)
	}

	// Verificar si el usuario está activo
	if !user.IsActive {
		return nil, ErrUserInactive
	}

	// Verificar la contraseña
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	// Generar tokens
	log.Printf("🔍 DEBUG Login: Calling generateTokenPair for user %s", user.Email)
	tokenPair, err := s.generateTokenPair(&user)
	if err != nil {
		log.Printf("❌ DEBUG Login: generateTokenPair failed: %v", err)
		return nil, fmt.Errorf("error al generar tokens: %w", err)
	}

	log.Printf("✅ DEBUG Login: generateTokenPair succeeded")
	return tokenPair, nil
}

// ValidateToken valida un token JWT y retorna los claims
func (s *service) ValidateToken(tokenString string) (*Claims, error) {
	// Parsear el token
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Verificar el método de firma
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("método de firma inesperado: %v", token.Header["alg"])
		}
		return []byte(s.config.JWT.Secret), nil
	})

	// Manejar errores de parseo
	if err != nil {
		return nil, fmt.Errorf("error al parsear token: %w", err)
	}

	// Verificar que el token sea válido
	if !token.Valid {
		return nil, ErrInvalidToken
	}

	// Extraer los claims
	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// RefreshToken actualiza el par de tokens usando un token de refresco
func (s *service) RefreshToken(refreshToken string) (*TokenPair, error) {
	// Validar el token de refresco
	claims, err := s.ValidateToken(refreshToken)
	if err != nil {
		return nil, err
	}

	// Obtener el usuario
	user, err := s.GetUserByID(claims.UserID)
	if err != nil {
		return nil, err
	}

	// Verificar si el usuario está activo
	if !user.IsActive {
		return nil, ErrUserInactive
	}

	// Generar un nuevo par de tokens
	tokenPair, err := s.generateTokenPair(user)
	if err != nil {
		return nil, fmt.Errorf("error al generar tokens: %w", err)
	}

	return tokenPair, nil
}

// GetUserByID obtiene un usuario por su ID
func (s *service) GetUserByID(id uuid.UUID) (*models.User, error) {
	var user models.User
	if err := s.db.Where("user_id = ?", id).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("usuario no encontrado")
		}
		return nil, fmt.Errorf("error al buscar usuario: %w", err)
	}
	return &user, nil
}

// generateTokenPair genera un par de tokens JWT (acceso y refresco)
func (s *service) generateTokenPair(user *models.User) (*TokenPair, error) {
	// Tiempo actual
	now := time.Now()
	
	// DEBUG: Mostrar configuración de JWT
	log.Printf("🔍 DEBUG generateTokenPair: AccessTokenExp=%v, RefreshTokenExp=%v", 
		s.config.JWT.AccessTokenExp, s.config.JWT.RefreshTokenExp)

	// Crear claims para el token de acceso
	accessClaims := Claims{
		UserID:   user.UserID,
		Email:    user.Email,
		UserRole: user.UserRole,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(s.config.JWT.AccessTokenExp)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "exactogas-api",
			Subject:   user.UserID.String(),
		},
	}

	// Crear token de acceso
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString([]byte(s.config.JWT.Secret))
	if err != nil {
		return nil, fmt.Errorf("error al firmar token de acceso: %w", err)
	}

	// Crear claims para el token de refresco
	refreshExpiry := now.Add(s.config.JWT.RefreshTokenExp)
	log.Printf("🔍 DEBUG refresh token: now=%v, expiry=%v, duration=%v", 
		now, refreshExpiry, s.config.JWT.RefreshTokenExp)
		
	refreshClaims := Claims{
		UserID:   user.UserID,
		Email:    user.Email,
		UserRole: user.UserRole,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(refreshExpiry),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "exactogas-api",
			Subject:   user.UserID.String(),
		},
	}

	// Crear token de refresco
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString([]byte(s.config.JWT.Secret))
	if err != nil {
		return nil, fmt.Errorf("error al firmar token de refresco: %w", err)
	}

	// Calcular segundos hasta la expiración
	expiresIn := int64(s.config.JWT.AccessTokenExp.Seconds())

	return &TokenPair{
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
		ExpiresIn:    expiresIn,
	}, nil
}
