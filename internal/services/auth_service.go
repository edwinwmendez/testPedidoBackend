package services

import (
	"backend/internal/models"
	"backend/internal/repositories"
)

type AuthService struct {
	userRepo repositories.UserRepository
}

func NewAuthService(userRepo repositories.UserRepository) *AuthService {
	return &AuthService{userRepo: userRepo}
}

func (s *AuthService) Login(email, password string) (*models.User, error) {
	// Lógica de autenticación
	return nil, nil
}
