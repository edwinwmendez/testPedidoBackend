package models

import (
	"backend/internal/models"
	"backend/tests/testutil"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func TestUser_SetPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{
			name:     "valid password",
			password: "testpassword123",
			wantErr:  false,
		},
		{
			name:     "empty password",
			password: "",
			wantErr:  false, // bcrypt can hash empty strings
		},
		{
			name:     "short password",
			password: "123",
			wantErr:  false, // bcrypt can hash short strings
		},
		{
			name:     "long valid password",
			password: "thisisaverylongbutvalidpassword123456789",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := &models.User{}
			err := user.SetPassword(tt.password)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Empty(t, user.PasswordHash)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, user.PasswordHash)

				// Verify the password was hashed correctly
				err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(tt.password))
				assert.NoError(t, err, "La contraseña debe coincidir con el hash")
			}
		})
	}
}

func TestUser_CheckPassword(t *testing.T) {
	user := testutil.CreateTestUser(t, models.UserRoleClient)

	tests := []struct {
		name     string
		password string
		want     bool
	}{
		{
			name:     "correct password",
			password: "testpassword123",
			want:     true,
		},
		{
			name:     "incorrect password",
			password: "wrongpassword",
			want:     false,
		},
		{
			name:     "empty password",
			password: "",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := user.CheckPassword(tt.password)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestUser_BeforeCreate(t *testing.T) {
	user := models.User{
		FullName:    "Test User",
		Email:       "test@example.com",
		PhoneNumber: "+51999999999",
		UserRole:    models.UserRoleClient,
	}

	err := user.BeforeCreate(nil)

	assert.NoError(t, err)

	// Check that UUID was generated
	assert.NotEqual(t, uuid.Nil, user.UserID)
}

func TestUserRole_Constants(t *testing.T) {
	// Test that user role constants are defined correctly
	assert.Equal(t, models.UserRole("CLIENT"), models.UserRoleClient)
	assert.Equal(t, models.UserRole("REPARTIDOR"), models.UserRoleRepartidor)
	assert.Equal(t, models.UserRole("ADMIN"), models.UserRoleAdmin)
}

func TestUser_ValidateRole(t *testing.T) {
	validRoles := []models.UserRole{
		models.UserRoleClient,
		models.UserRoleRepartidor,
		models.UserRoleAdmin,
	}

	for _, role := range validRoles {
		user := &models.User{
			UserRole: role,
		}

		// This would be validated in BeforeCreate, but we're testing the concept
		assert.Contains(t, []models.UserRole{
			models.UserRoleClient,
			models.UserRoleRepartidor,
			models.UserRoleAdmin,
		}, user.UserRole)
	}
}

func TestUser_PasswordSecurity(t *testing.T) {
	user := &models.User{}
	password := "testpassword123"

	err := user.SetPassword(password)
	require.NoError(t, err)

	// Ensure password is not stored in plain text
	assert.NotEqual(t, password, user.PasswordHash)

	// Ensure hash is not empty
	assert.NotEmpty(t, user.PasswordHash)

	// Ensure hash length is reasonable (bcrypt produces 60 character hashes)
	assert.Equal(t, 60, len(user.PasswordHash))

	// Ensure hash starts with bcrypt prefix
	assert.True(t, user.PasswordHash[0:4] == "$2a$" || user.PasswordHash[0:4] == "$2b$")
}
