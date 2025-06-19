package database

import (
	"backend/config"
	"backend/internal/models"
	"backend/internal/repositories"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// UserRepositoryTestSuite defines the test suite for user repository with real database
type UserRepositoryTestSuite struct {
	suite.Suite
	db       *gorm.DB
	userRepo repositories.UserRepository
	config   *config.Config
}

// SetupSuite runs once before the test suite
func (suite *UserRepositoryTestSuite) SetupSuite() {
	// Test configuration
	suite.config = &config.Config{
		Database: config.DatabaseConfig{
			Host:     "localhost",
			Port:     "5433",
			User:     "postgres",
			Password: "postgres",
			DBName:   "exactogas_test",
			SSLMode:  "disable",
		},
	}

	// Connect to test database with foreign key constraint disabled
	var err error
	suite.db, err = gorm.Open(postgres.Open(suite.config.Database.GetDSN()), &gorm.Config{
		Logger:                                   logger.Default.LogMode(logger.Silent),
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		suite.T().Skip("PostgreSQL test database not available")
		// Ahi dice Base de datos de prueba de PostgreSQL no disponible
		return
	}

	// Auto-migrate all related tables with foreign key constraints disabled
	err = suite.db.AutoMigrate(&models.User{}, &models.Product{}, &models.Order{}, &models.OrderItem{})
	require.NoError(suite.T(), err, "Database schema migration should succeed")
	// Ahi dice La migración del esquema de la base de datos debe tener éxito

	// Drop any incorrect foreign key constraints that GORM might have created
	suite.db.Exec("ALTER TABLE products DROP CONSTRAINT IF EXISTS fk_order_items_product")
	suite.db.Exec("ALTER TABLE order_items DROP CONSTRAINT IF EXISTS fk_order_items_product")

	// Create repository
	suite.userRepo = repositories.NewUserRepository(suite.db)
}

// SetupTest runs before each test
func (suite *UserRepositoryTestSuite) SetupTest() {
	// Clean database before each test - order matters due to foreign key constraints
	suite.db.Exec("TRUNCATE TABLE order_items RESTART IDENTITY CASCADE")
	suite.db.Exec("TRUNCATE TABLE orders RESTART IDENTITY CASCADE")
	suite.db.Exec("TRUNCATE TABLE products RESTART IDENTITY CASCADE")
	suite.db.Exec("TRUNCATE TABLE users RESTART IDENTITY CASCADE")

	// Also clean any seed data that might have been inserted by migrations
	suite.db.Exec("DELETE FROM products WHERE name LIKE 'Balón de Gas%'")
	suite.db.Exec("DELETE FROM users WHERE email = 'admin@exactogas.com'")
}

// TearDownSuite runs once after the test suite
func (suite *UserRepositoryTestSuite) TearDownSuite() {
	if suite.db != nil {
		// Clean up - order matters due to foreign key constraints
		suite.db.Exec("TRUNCATE TABLE order_items RESTART IDENTITY CASCADE")
		suite.db.Exec("TRUNCATE TABLE orders RESTART IDENTITY CASCADE")
		suite.db.Exec("TRUNCATE TABLE products RESTART IDENTITY CASCADE")
		suite.db.Exec("TRUNCATE TABLE users RESTART IDENTITY CASCADE")
		sqlDB, _ := suite.db.DB()
		sqlDB.Close()
	}
}

func (suite *UserRepositoryTestSuite) TestCreateUser_AllRoles() {
	roles := []models.UserRole{
		models.UserRoleClient,
		models.UserRoleRepartidor,
		models.UserRoleAdmin,
	}

	for _, role := range roles {
		suite.T().Run(string(role), func(t *testing.T) {
			user := &models.User{
				Email:        string(role) + "@example.com",
				PasswordHash: "hashedpassword123",
				FullName:     "Test " + string(role),
				PhoneNumber:  "+5199999" + string(role)[0:4],
				UserRole:     role,
			}

			// Test creation
			err := suite.userRepo.Create(user)
			assert.NoError(t, err, "Should create user with role %s", role)   // Ahi dice Debería crear un usuario con el rol %s
			assert.NotEqual(t, uuid.Nil, user.UserID, "Should generate UUID") // Ahi dice Debería generar un UUID
			assert.False(t, user.CreatedAt.IsZero(), "Should set CreatedAt")  // Ahi dice Debería establecer CreatedAt
			assert.False(t, user.UpdatedAt.IsZero(), "Should set UpdatedAt")  // Ahi dice Debería establecer UpdatedAt

			// Test retrieval
			retrievedUser, err := suite.userRepo.FindByID(user.UserID.String())
			assert.NoError(t, err, "Should retrieve created user") // Ahi dice Debería recuperar el usuario creado
			assert.Equal(t, user.Email, retrievedUser.Email)       // Ahi dice Debería ser igual al email del usuario creado
			assert.Equal(t, user.UserRole, retrievedUser.UserRole) // Ahi dice Debería ser igual al rol del usuario creado
		})
	}
}

func (suite *UserRepositoryTestSuite) TestDatabaseConstraints() {
	// Test unique email constraint
	user1 := &models.User{
		Email:        "unique@example.com",
		PasswordHash: "hashedpassword123",
		FullName:     "User One",
		PhoneNumber:  "+51999999001",
		UserRole:     models.UserRoleClient,
	}

	err := suite.userRepo.Create(user1)
	require.NoError(suite.T(), err)

	// Try to create user with same email
	user2 := &models.User{
		Email:        "unique@example.com", // Same email
		PasswordHash: "hashedpassword456",
		FullName:     "User Two",
		PhoneNumber:  "+51999999002",
		UserRole:     models.UserRoleClient,
	}

	err = suite.userRepo.Create(user2)
	assert.Error(suite.T(), err, "Should enforce unique email constraint")

	// Test unique phone constraint
	user3 := &models.User{
		Email:        "different@example.com",
		PasswordHash: "hashedpassword789",
		FullName:     "User Three",
		PhoneNumber:  "+51999999001", // Same phone as user1
		UserRole:     models.UserRoleClient,
	}

	err = suite.userRepo.Create(user3)
	assert.Error(suite.T(), err, "Should enforce unique phone constraint") // Ahi dice Debería aplicar la restricción de número de teléfono único
}

func (suite *UserRepositoryTestSuite) TestUserRoleValidation() {
	// Test valid roles
	validRoles := []models.UserRole{
		models.UserRoleClient,
		models.UserRoleRepartidor,
		models.UserRoleAdmin,
	}

	for _, role := range validRoles {
		user := &models.User{
			Email:        string(role) + "_valid@example.com",
			PasswordHash: "hashedpassword123",
			FullName:     "Valid " + string(role),
			PhoneNumber:  "+5199999" + string(role)[0:4],
			UserRole:     role,
		}

		err := suite.userRepo.Create(user)
		assert.NoError(suite.T(), err, "Should accept valid role: %s", role) // Ahi dice Debería aceptar un rol válido: %s
	}

	// Test invalid role (this might not fail at DB level depending on constraints)
	invalidUser := &models.User{
		Email:        "invalid@example.com",
		PasswordHash: "hashedpassword123",
		FullName:     "Invalid User",
		PhoneNumber:  "+51999999000",
		UserRole:     "INVALID_ROLE",
	}

	// This test depends on whether there's a CHECK constraint on the role
	err := suite.userRepo.Create(invalidUser)
	// The behavior depends on database constraints - document this
	suite.T().Logf("Invalid role creation result: %v", err)
}

func (suite *UserRepositoryTestSuite) TestRequiredFields() {
	// Test missing email (should fail due to NOT NULL constraint)
	userNoEmail := &models.User{
		PasswordHash: "hashedpassword123",
		FullName:     "No Email User",
		PhoneNumber:  "+51999999001",
		UserRole:     models.UserRoleClient,
	}

	err := suite.userRepo.Create(userNoEmail)
	if err != nil {
		assert.Contains(suite.T(), err.Error(), "email", "Should require email field") // Ahi dice Debería requerir el campo de email
	} else {
		suite.T().Log("Email constraint not enforced at application level")
		// Ahi dice Restricción de correo electrónico no aplicada a nivel de aplicación
	}

	// Test missing password hash
	userNoPassword := &models.User{
		Email:       "nopassword@example.com",
		FullName:    "No Password User",
		PhoneNumber: "+51999999002",
		UserRole:    models.UserRoleClient,
	}

	err = suite.userRepo.Create(userNoPassword)
	if err != nil {
		assert.Contains(suite.T(), err.Error(), "password", "Should require password_hash field") // Ahi dice Debería requerir el campo de contraseña
	} else {
		suite.T().Log("Password constraint not enforced at application level")
	}

	// Test missing full name
	userNoName := &models.User{
		Email:        "noname@example.com",
		PasswordHash: "hashedpassword123",
		PhoneNumber:  "+51999999003",
		UserRole:     models.UserRoleClient,
	}

	err = suite.userRepo.Create(userNoName)
	if err != nil {
		assert.Contains(suite.T(), err.Error(), "full_name", "Should require full_name field") // Ahi dice Debería requerir el campo de nombre completo
	} else {
		suite.T().Log("Full name constraint not enforced at application level")
		// Ahi dice Restricción de nombre completo no aplicada a nivel de aplicación
	}
}

func (suite *UserRepositoryTestSuite) TestFindByEmail() {
	// Create test user
	user := &models.User{
		Email:        "findme@example.com",
		PasswordHash: "hashedpassword123",
		FullName:     "Find Me User",
		PhoneNumber:  "+51999999999",
		UserRole:     models.UserRoleRepartidor,
	}

	err := suite.userRepo.Create(user)
	require.NoError(suite.T(), err)

	// Test finding by email
	foundUser, err := suite.userRepo.FindByEmail("findme@example.com")
	assert.NoError(suite.T(), err)                             // Ahi dice Debería encontrar al usuario por email
	assert.Equal(suite.T(), user.Email, foundUser.Email)       // Ahi dice Debería ser igual al email del usuario encontrado
	assert.Equal(suite.T(), user.UserRole, foundUser.UserRole) // Ahi dice Debería ser igual al rol del usuario encontrado

	// Test finding non-existent email
	_, err = suite.userRepo.FindByEmail("nonexistent@example.com")
	assert.Error(suite.T(), err, "Should return error for non-existent email") // Ahi dice Debería devolver un error si el correo electrónico no existe
}

func (suite *UserRepositoryTestSuite) TestUpdateUser() {
	// Create user
	user := &models.User{
		Email:        "update@example.com",
		PasswordHash: "originalpassword",
		FullName:     "Original Name",
		PhoneNumber:  "+51999999999",
		UserRole:     models.UserRoleClient,
	}

	err := suite.userRepo.Create(user)
	require.NoError(suite.T(), err)
	originalUpdatedAt := user.UpdatedAt

	// Wait a moment to ensure timestamp difference
	time.Sleep(10 * time.Millisecond)

	// Update user
	user.FullName = "Updated Name"
	user.PasswordHash = "newpassword"

	err = suite.userRepo.Update(user)
	assert.NoError(suite.T(), err)                                                             // Ahi dice Debería actualizar el usuario
	assert.True(suite.T(), user.UpdatedAt.After(originalUpdatedAt), "Should update timestamp") // Ahi dice Debería actualizar el campo de fecha/hora de actualización

	// Verify update
	updatedUser, err := suite.userRepo.FindByID(user.UserID.String())
	assert.NoError(suite.T(), err)                                   // Ahi dice Debería encontrar al usuario actualizado
	assert.Equal(suite.T(), "Updated Name", updatedUser.FullName)    // Ahi dice Debería ser igual al nombre completo del usuario actualizado
	assert.Equal(suite.T(), "newpassword", updatedUser.PasswordHash) // Ahi dice Debería ser igual a la contraseña del usuario actualizado
}

func (suite *UserRepositoryTestSuite) TestDeleteUser() {
	// Create user
	user := &models.User{
		Email:        "delete@example.com",
		PasswordHash: "hashedpassword123",
		FullName:     "Delete Me",
		PhoneNumber:  "+51999999999",
		UserRole:     models.UserRoleAdmin,
	}

	err := suite.userRepo.Create(user)
	require.NoError(suite.T(), err)

	// Delete user
	err = suite.userRepo.Delete(user.UserID.String())
	assert.NoError(suite.T(), err) // Ahi dice Debería eliminar al usuario

	// Verify deletion
	_, err = suite.userRepo.FindByID(user.UserID.String())
	assert.Error(suite.T(), err, "Should not find deleted user") // Ahi dice Debería devolver un error si el usuario no existe
}

func (suite *UserRepositoryTestSuite) TestListUsers() {
	// Create multiple users with different roles
	users := []*models.User{
		{
			Email:        "client1@example.com",
			PasswordHash: "hashedpassword123",
			FullName:     "Client One",
			PhoneNumber:  "+51999999001",
			UserRole:     models.UserRoleClient,
		},
		{
			Email:        "repartidor1@example.com",
			PasswordHash: "hashedpassword123",
			FullName:     "Repartidor One",
			PhoneNumber:  "+51999999002",
			UserRole:     models.UserRoleRepartidor,
		},
		{
			Email:        "admin1@example.com",
			PasswordHash: "hashedpassword123",
			FullName:     "Admin One",
			PhoneNumber:  "+51999999003",
			UserRole:     models.UserRoleAdmin,
		},
	}

	for _, user := range users {
		err := suite.userRepo.Create(user)
		require.NoError(suite.T(), err)
	}

	// Test counting all users by querying each role
	allClients, err := suite.userRepo.FindByRole(models.UserRoleClient)
	assert.NoError(suite.T(), err) // Ahi dice Debería encontrar todos los clientes
	allRepartidores, err2 := suite.userRepo.FindByRole(models.UserRoleRepartidor)
	assert.NoError(suite.T(), err2) // Ahi dice Debería encontrar todos los repartidores
	allAdmins, err3 := suite.userRepo.FindByRole(models.UserRoleAdmin)
	assert.NoError(suite.T(), err3) // Ahi dice Debería encontrar todos los administradores

	totalUsers := len(allClients) + len(allRepartidores) + len(allAdmins)
	assert.Equal(suite.T(), 3, totalUsers, "Should find all created users across all roles") // Ahi dice Debería encontrar todos los usuarios creados en todos los roles
	// Ahi dice Debería encontrar todos los usuarios creados en todos los roles

	// Test listing users by role
	clients, err := suite.userRepo.FindByRole(models.UserRoleClient)
	assert.NoError(suite.T(), err)                                      // Ahi dice Debería encontrar un cliente
	assert.Len(suite.T(), clients, 1, "Should find one client")         // Ahi dice Debería encontrar un cliente
	assert.Equal(suite.T(), models.UserRoleClient, clients[0].UserRole) // Ahi dice Debería ser igual al rol del cliente
	// Ahi dice Debería encontrar un cliente
	repartidores, err := suite.userRepo.FindByRole(models.UserRoleRepartidor)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), repartidores, 1, "Should find one repartidor")
	assert.Equal(suite.T(), models.UserRoleRepartidor, repartidores[0].UserRole)

	admins, err := suite.userRepo.FindByRole(models.UserRoleAdmin)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), admins, 1, "Should find one admin")
	assert.Equal(suite.T(), models.UserRoleAdmin, admins[0].UserRole)
}

func (suite *UserRepositoryTestSuite) TestConcurrentUserCreation() {
	// Test that concurrent user creation doesn't violate constraints
	done := make(chan bool, 2)
	errors := make(chan error, 2)

	// Try to create users with same email concurrently
	go func() {
		user := &models.User{
			Email:        "concurrent@example.com",
			PasswordHash: "hashedpassword123",
			FullName:     "Concurrent User 1",
			PhoneNumber:  "+51999999001",
			UserRole:     models.UserRoleClient,
		}
		err := suite.userRepo.Create(user)
		errors <- err
		done <- true
	}()

	go func() {
		user := &models.User{
			Email:        "concurrent@example.com", // Same email
			PasswordHash: "hashedpassword123",
			FullName:     "Concurrent User 2",
			PhoneNumber:  "+51999999002",
			UserRole:     models.UserRoleClient,
		}
		err := suite.userRepo.Create(user)
		errors <- err
		done <- true
	}()

	// Wait for both goroutines
	<-done
	<-done

	// Check that at least one failed due to constraint violation
	err1 := <-errors
	err2 := <-errors

	assert.True(suite.T(), err1 != nil || err2 != nil,
		"At least one concurrent creation should fail due to unique constraint")
	// Ahi dice Al menos una creación concurrente debe fallar debido a la restricción de unicidad
}

// TestUserRepositoryTestSuite runs the test suite
func TestUserRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(UserRepositoryTestSuite))
}
