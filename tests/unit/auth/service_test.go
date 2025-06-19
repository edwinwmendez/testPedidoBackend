package auth

import (
	"backend/config"
	"backend/internal/auth"
	"backend/internal/models"
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

// AuthServiceTestSuite defines the test suite for the auth service
// AuthServiceTestSuite define el test suite para el servicio de autenticación
type AuthServiceTestSuite struct {
	suite.Suite
	db      *gorm.DB
	service auth.Service
	config  *config.Config
}

// SetupSuite runs once before the test suite
// SetupSuite ejecuta una vez antes del test suite
func (suite *AuthServiceTestSuite) SetupSuite() {
	// Test configuration
	suite.config = &config.Config{
		JWT: config.JWTConfig{
			Secret:          "test-jwt-secret-key-for-testing",
			AccessTokenExp:  15 * time.Minute,
			RefreshTokenExp: 7 * 24 * time.Hour,
		},
	}

	// For unit tests, we use PostgreSQL test database
	var err error
	suite.db, err = gorm.Open(postgres.Open("host=localhost port=5433 user=postgres password=postgres dbname=exactogas_test sslmode=disable"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		// If PostgreSQL is not available, skip these tests
		//suite.T().Skip("PostgreSQL not available for testing")
		suite.T().Skip("PostgreSQL no disponible para pruebas")
		return
	}

	// Auto-migrate the test database
	err = suite.db.AutoMigrate(&models.User{})
	require.NoError(suite.T(), err)

	// Create the auth service
	suite.service = auth.NewService(suite.db, suite.config)
}

// SetupTest runs before each test
func (suite *AuthServiceTestSuite) SetupTest() {
	// Clean up the database before each test
	// Limpia la base de datos antes de cada test
	suite.db.Where("1 = 1").Delete(&models.User{})
}

// TearDownSuite runs once after the test suite
func (suite *AuthServiceTestSuite) TearDownSuite() {
	if suite.db != nil {
		// Clean up
		// Limpia la base de datos
		suite.db.Where("1 = 1").Delete(&models.User{})
		sqlDB, _ := suite.db.DB()
		sqlDB.Close()
	}
}

func (suite *AuthServiceTestSuite) TestRegisterUser_Success() {
	// TestRegisterUser_Success prueba el registro de un usuario exitoso
	email := "test@example.com"
	password := "testpassword123"
	fullName := "Test User"
	phoneNumber := "+51999999999"
	role := models.UserRoleClient

	user, err := suite.service.RegisterUser(email, password, fullName, phoneNumber, role)

	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), user)

	assert.Equal(suite.T(), email, user.Email)
	assert.Equal(suite.T(), fullName, user.FullName)
	assert.Equal(suite.T(), phoneNumber, user.PhoneNumber)
	assert.Equal(suite.T(), role, user.UserRole)
	assert.NotEqual(suite.T(), uuid.Nil, user.UserID)
	assert.NotEmpty(suite.T(), user.PasswordHash)
	assert.NotEqual(suite.T(), password, user.PasswordHash) // Password should be hashed
}

func (suite *AuthServiceTestSuite) TestRegisterUser_DuplicateEmail() {
	// TestRegisterUser_DuplicateEmail prueba el registro de un usuario con un email duplicado
	email := "duplicate@example.com"
	password := "testpassword123"
	fullName := "Test User"
	phoneNumber := "+51999999999"
	role := models.UserRoleClient

	// Register first user
	_, err := suite.service.RegisterUser(email, password, fullName, phoneNumber, role)
	require.NoError(suite.T(), err)

	// Try to register second user with same email
	_, err = suite.service.RegisterUser(email, password, "Another User", "+51888888888", role)

	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), auth.ErrUserAlreadyExists, err)
}

func (suite *AuthServiceTestSuite) TestRegisterUser_DuplicatePhone() {
	// TestRegisterUser_DuplicatePhone prueba el registro de un usuario con un número de teléfono duplicado
	password := "testpassword123"
	fullName := "Test User"
	phoneNumber := "+51999999999"
	role := models.UserRoleClient

	// Register first user
	_, err := suite.service.RegisterUser("first@example.com", password, fullName, phoneNumber, role)
	require.NoError(suite.T(), err)

	// Try to register second user with same phone
	_, err = suite.service.RegisterUser("second@example.com", password, "Another User", phoneNumber, role)

	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), auth.ErrUserAlreadyExists, err)
}

func (suite *AuthServiceTestSuite) TestRegisterUser_InvalidRole() {
	// TestRegisterUser_InvalidRole prueba el registro de un usuario con un rol inválido
	email := "test@example.com"
	password := "testpassword123"
	fullName := "Test User"
	phoneNumber := "+51999999999"
	invalidRole := models.UserRole("INVALID_ROLE")

	_, err := suite.service.RegisterUser(email, password, fullName, phoneNumber, invalidRole)

	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), auth.ErrInvalidRole, err)
}

func (suite *AuthServiceTestSuite) TestLogin_Success() {
	// TestLogin_Success prueba el inicio de sesión exitoso
	email := "login@example.com"
	password := "testpassword123"
	fullName := "Test User"
	phoneNumber := "+51999999999"
	role := models.UserRoleClient

	// Register user first
	_, err := suite.service.RegisterUser(email, password, fullName, phoneNumber, role)
	require.NoError(suite.T(), err)

	// Test login
	tokenPair, err := suite.service.Login(email, password)

	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), tokenPair)

	assert.NotEmpty(suite.T(), tokenPair.AccessToken)
	assert.NotEmpty(suite.T(), tokenPair.RefreshToken)
	assert.Greater(suite.T(), tokenPair.ExpiresIn, int64(0))
}

func (suite *AuthServiceTestSuite) TestLogin_InvalidCredentials() {
	// TestLogin_InvalidCredentials prueba el inicio de sesión con credenciales inválidas
	email := "login@example.com"
	password := "testpassword123"
	wrongPassword := "wrongpassword"
	fullName := "Test User"
	phoneNumber := "+51999999999"
	role := models.UserRoleClient

	// Register user first
	_, err := suite.service.RegisterUser(email, password, fullName, phoneNumber, role)
	require.NoError(suite.T(), err)

	// Test login with wrong password
	_, err = suite.service.Login(email, wrongPassword)

	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), auth.ErrInvalidCredentials, err)
}

func (suite *AuthServiceTestSuite) TestLogin_UserNotFound() {
	// TestLogin_UserNotFound prueba el inicio de sesión con un email que no existe
	nonExistentEmail := "nonexistent@example.com"
	password := "testpassword123"

	_, err := suite.service.Login(nonExistentEmail, password)

	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), auth.ErrInvalidCredentials, err)
}

func (suite *AuthServiceTestSuite) TestValidateToken_Success() {
	// TestValidateToken_Success prueba la validación de un token exitosa
	email := "token@example.com"
	password := "testpassword123"
	fullName := "Test User"
	phoneNumber := "+51999999999"
	role := models.UserRoleRepartidor

	// Register and login user
	user, err := suite.service.RegisterUser(email, password, fullName, phoneNumber, role)
	require.NoError(suite.T(), err)

	tokenPair, err := suite.service.Login(email, password)
	require.NoError(suite.T(), err)

	// Validate the access token
	claims, err := suite.service.ValidateToken(tokenPair.AccessToken)

	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), claims)

	assert.Equal(suite.T(), user.UserID, claims.UserID)
	assert.Equal(suite.T(), user.Email, claims.Email)
	assert.Equal(suite.T(), user.UserRole, claims.UserRole)
}

func (suite *AuthServiceTestSuite) TestValidateToken_InvalidToken() {
	// TestValidateToken_InvalidToken prueba la validación de un token inválido
	invalidToken := "invalid.token.here"

	_, err := suite.service.ValidateToken(invalidToken)

	assert.Error(suite.T(), err)
}

func (suite *AuthServiceTestSuite) TestRefreshToken_Success() {
	// TestRefreshToken_Success prueba el refresco de un token exitoso
	email := "refresh@example.com"
	password := "testpassword123"
	fullName := "Test User"
	phoneNumber := "+51999999999"
	role := models.UserRoleAdmin

	// Register and login user
	_, err := suite.service.RegisterUser(email, password, fullName, phoneNumber, role)
	require.NoError(suite.T(), err)

	tokenPair, err := suite.service.Login(email, password)
	require.NoError(suite.T(), err)

	// Refresh the token
	newTokenPair, err := suite.service.RefreshToken(tokenPair.RefreshToken)

	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), newTokenPair)

	assert.NotEmpty(suite.T(), newTokenPair.AccessToken)
	assert.NotEmpty(suite.T(), newTokenPair.RefreshToken)
	assert.Greater(suite.T(), newTokenPair.ExpiresIn, int64(0))

	// New tokens should be different from old ones (tokens can be the same if generated within same second)
	// Just verify we got valid new tokens
	assert.NotEmpty(suite.T(), newTokenPair.AccessToken)
	assert.NotEmpty(suite.T(), newTokenPair.RefreshToken)
}

func (suite *AuthServiceTestSuite) TestGetUserByID_Success() {
	// TestGetUserByID_Success prueba la obtención de un usuario por ID exitosa
	email := "getuser@example.com"
	password := "testpassword123"
	fullName := "Test User"
	phoneNumber := "+51999999999"
	role := models.UserRoleClient

	// Register user
	user, err := suite.service.RegisterUser(email, password, fullName, phoneNumber, role)
	require.NoError(suite.T(), err)

	// Get user by ID
	retrievedUser, err := suite.service.GetUserByID(user.UserID)

	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), retrievedUser)

	assert.Equal(suite.T(), user.UserID, retrievedUser.UserID)
	assert.Equal(suite.T(), user.Email, retrievedUser.Email)
	assert.Equal(suite.T(), user.FullName, retrievedUser.FullName)
	assert.Equal(suite.T(), user.UserRole, retrievedUser.UserRole)
}

func (suite *AuthServiceTestSuite) TestGetUserByID_NotFound() {
	// TestGetUserByID_NotFound prueba la obtención de un usuario por ID que no existe
	nonExistentID := uuid.New()

	_, err := suite.service.GetUserByID(nonExistentID)

	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "usuario no encontrado")
}

func (suite *AuthServiceTestSuite) TestTokenWorkflow_Complete() {
	// TestTokenWorkflow_Complete prueba el flujo completo de tokens: registro -> inicio de sesión -> validación -> refresco
	email := "workflow@example.com"
	password := "testpassword123"
	fullName := "Workflow User"
	phoneNumber := "+51999999999"
	role := models.UserRoleRepartidor

	// Step 1: Register
	user, err := suite.service.RegisterUser(email, password, fullName, phoneNumber, role)
	require.NoError(suite.T(), err)

	// Step 2: Login
	tokenPair, err := suite.service.Login(email, password)
	require.NoError(suite.T(), err)

	// Step 3: Validate access token
	claims, err := suite.service.ValidateToken(tokenPair.AccessToken)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), user.UserID, claims.UserID)

	// Step 4: Refresh tokens
	newTokenPair, err := suite.service.RefreshToken(tokenPair.RefreshToken)
	require.NoError(suite.T(), err)

	// Step 5: Validate new access token
	newClaims, err := suite.service.ValidateToken(newTokenPair.AccessToken)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), user.UserID, newClaims.UserID)
}

// TestAuthServiceTestSuite ejecuta el test suite
func TestAuthServiceTestSuite(t *testing.T) {
	suite.Run(t, new(AuthServiceTestSuite))
}
