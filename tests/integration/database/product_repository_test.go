package database

import (
	"backend/config"
	"backend/internal/models"
	"backend/internal/repositories"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ProductRepositoryTestSuite defines the test suite for product repository with real database
type ProductRepositoryTestSuite struct {
	suite.Suite
	db          *gorm.DB
	productRepo repositories.ProductRepository
	config      *config.Config
}

// SetupSuite runs once before the test suite
func (suite *ProductRepositoryTestSuite) SetupSuite() {
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
		//suite.T().Skip("PostgreSQL test database not available")
		suite.T().Skip("PostgreSQL no disponible para pruebas")
		return
	}

	// Auto-migrate all related tables with foreign key constraints disabled
	err = suite.db.AutoMigrate(&models.User{}, &models.Product{}, &models.Order{}, &models.OrderItem{})
	require.NoError(suite.T(), err, "La migración de la base de datos debe ser exitosa")

	// Drop any incorrect foreign key constraints that GORM might have created
	suite.db.Exec("ALTER TABLE products DROP CONSTRAINT IF EXISTS fk_order_items_product")
	suite.db.Exec("ALTER TABLE order_items DROP CONSTRAINT IF EXISTS fk_order_items_product")

	// Create repository
	suite.productRepo = repositories.NewProductRepository(suite.db)
}

// SetupTest runs before each test
func (suite *ProductRepositoryTestSuite) SetupTest() {
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
func (suite *ProductRepositoryTestSuite) TearDownSuite() {
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

func (suite *ProductRepositoryTestSuite) TestCreateProduct_Success() {
	product := &models.Product{
		ProductID:   uuid.New(),
		Name:        "Gas Balón 10kg",
		Description: "Balón de gas doméstico de 10 kilogramos",
		Price:       25.50,
		IsActive:    true,
	}

	err := suite.productRepo.Create(product)
	assert.NoError(suite.T(), err, "Debe crear el producto exitosamente")
	assert.NotEqual(suite.T(), uuid.Nil, product.ProductID, "Debe preservar el UUID")
	assert.False(suite.T(), product.CreatedAt.IsZero(), "Debe establecer CreatedAt")
	assert.False(suite.T(), product.UpdatedAt.IsZero(), "Debe establecer UpdatedAt")
}

func (suite *ProductRepositoryTestSuite) TestFindByID_Success() {
	// Create product
	product := &models.Product{
		ProductID:   uuid.New(),
		Name:        "Gas Balón 5kg",
		Description: "Balón de gas doméstico de 5 kilogramos",
		Price:       15.75,
		IsActive:    true,
	}

	err := suite.productRepo.Create(product)
	require.NoError(suite.T(), err)

	// Find by ID
	foundProduct, err := suite.productRepo.FindByID(product.ProductID.String())
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), product.ProductID, foundProduct.ProductID)
	assert.Equal(suite.T(), product.Name, foundProduct.Name)
	assert.Equal(suite.T(), product.Price, foundProduct.Price)
	assert.Equal(suite.T(), product.IsActive, foundProduct.IsActive)
}

func (suite *ProductRepositoryTestSuite) TestFindByID_NotFound() {
	nonExistentID := uuid.New()

	_, err := suite.productRepo.FindByID(nonExistentID.String())
	assert.Error(suite.T(), err, "Debe devolver un error para un producto inexistente")
}

func (suite *ProductRepositoryTestSuite) TestFindAll() {
	// Create multiple products
	products := []*models.Product{
		{
			ProductID:   uuid.New(),
			Name:        "Gas Balón 10kg",
			Description: "Balón de gas doméstico de 10 kilogramos",
			Price:       25.50,
			IsActive:    true,
		},
		{
			ProductID:   uuid.New(),
			Name:        "Gas Balón 15kg",
			Description: "Balón de gas industrial de 15 kilogramos",
			Price:       35.75,
			IsActive:    false,
		},
		{
			ProductID:   uuid.New(),
			Name:        "Gas Balón 5kg",
			Description: "Balón de gas doméstico de 5 kilogramos",
			Price:       15.25,
			IsActive:    true,
		},
	}

	for _, product := range products {
		err := suite.productRepo.Create(product)
		require.NoError(suite.T(), err)
	}

	// Find all products
	allProducts, err := suite.productRepo.FindAll()
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), allProducts, 3, "Debe encontrar todos los productos creados")
}

func (suite *ProductRepositoryTestSuite) TestFindActive() {
	// Create active products first
	activeProduct1 := &models.Product{
		ProductID:   uuid.New(),
		Name:        "Producto activo 1",
		Description: "Descripción del producto activo 1",
		Price:       25.50,
		IsActive:    true,
	}
	err := suite.productRepo.Create(activeProduct1)
	require.NoError(suite.T(), err)

	activeProduct2 := &models.Product{
		ProductID:   uuid.New(),
		Name:        "Producto activo 2",
		Description: "Descripción del producto activo 2",
		Price:       15.25,
		IsActive:    true,
	}
	err = suite.productRepo.Create(activeProduct2)
	require.NoError(suite.T(), err)

	// Create inactive product using repository (now fixed to handle false values)
	inactiveProduct := &models.Product{
		ProductID:   uuid.New(),
		Name:        "Producto inactivo",
		Description: "Descripción del producto inactivo",
		Price:       35.75,
		IsActive:    false,
	}
	err = suite.productRepo.Create(inactiveProduct)
	require.NoError(suite.T(), err)

	// Verify the inactive product was actually saved as inactive
	savedInactive, err := suite.productRepo.FindByID(inactiveProduct.ProductID.String())
	require.NoError(suite.T(), err)
	suite.T().Logf("Producto inactivo después de guardar: Name=%s, IsActive=%t", savedInactive.Name, savedInactive.IsActive)
	assert.False(suite.T(), savedInactive.IsActive, "El producto inactivo debe permanecer inactivo")

	// Find only active products
	activeProducts, err := suite.productRepo.FindActive()
	assert.NoError(suite.T(), err)

	// Debug: Print what products were found
	suite.T().Logf("Found %d active products:", len(activeProducts))
	for i, product := range activeProducts {
		suite.T().Logf("  Product %d: %s (ID: %s, Active: %t)", i+1, product.Name, product.ProductID, product.IsActive)
	}

	assert.Len(suite.T(), activeProducts, 2, "Debe encontrar solo productos activos")

	for _, product := range activeProducts {
		assert.True(suite.T(), product.IsActive, "Todos los productos devueltos deben ser activos")
	}
}

func (suite *ProductRepositoryTestSuite) TestUpdateProduct() {
	// Create product
	product := &models.Product{
		ProductID:   uuid.New(),
		Name:        "Nombre original",
		Description: "Descripción original",
		Price:       25.50,
		IsActive:    true,
	}

	err := suite.productRepo.Create(product)
	require.NoError(suite.T(), err)
	originalUpdatedAt := product.UpdatedAt

	// Update product
	product.Name = "Nombre actualizado"
	product.Price = 35.75
	product.IsActive = false
	product.Description = "Descripción actualizada"

	err = suite.productRepo.Update(product)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), product.UpdatedAt.After(originalUpdatedAt), "Debe actualizar la marca de tiempo")

	// Verify update
	updatedProduct, err := suite.productRepo.FindByID(product.ProductID.String())
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "Nombre actualizado", updatedProduct.Name)
	assert.Equal(suite.T(), 35.75, updatedProduct.Price)
	assert.False(suite.T(), updatedProduct.IsActive)
	assert.Equal(suite.T(), "Descripción actualizada", updatedProduct.Description)
}

func (suite *ProductRepositoryTestSuite) TestDeleteProduct() {
	// Create product
	product := &models.Product{
		ProductID:   uuid.New(),
		Name:        "Producto a eliminar",
		Description: "Este producto será eliminado",
		Price:       25.50,
		IsActive:    true,
	}

	err := suite.productRepo.Create(product)
	require.NoError(suite.T(), err)

	// Delete product
	err = suite.productRepo.Delete(product.ProductID.String())
	assert.NoError(suite.T(), err)

	// Verify deletion
	_, err = suite.productRepo.FindByID(product.ProductID.String())
	assert.Error(suite.T(), err, "No debe encontrar el producto eliminado")
}

func (suite *ProductRepositoryTestSuite) TestProductConstraints() {
	// Test required fields
	productNoName := &models.Product{
		ProductID:   uuid.New(),
		Description: "Producto sin nombre",
		Price:       25.50,
		IsActive:    true,
	}

	err := suite.productRepo.Create(productNoName)
	// Depending on database constraints, this might pass or fail
	// Document the behavior
	if err != nil {
		assert.Contains(suite.T(), err.Error(), "name", "Debe requerir el campo name")
		suite.T().Log("La restricción de nombre se aplica en el nivel de base de datos")
	} else {
		suite.T().Log("La restricción de nombre no se aplica en el nivel de base de datos")
	}

	// Test negative price
	productNegativePrice := &models.Product{
		ProductID:   uuid.New(),
		Name:        "Producto con precio negativo",
		Description: "Producto con precio negativo",
		Price:       -10.00,
		IsActive:    true,
	}

	err = suite.productRepo.Create(productNegativePrice)
	// This should ideally be constrained at database level
	if err != nil {
		assert.Contains(suite.T(), err.Error(), "price", "Debe validar la restricción de precio")
		suite.T().Log("La restricción de precio se aplica en el nivel de base de datos")
	} else {
		suite.T().Log("La restricción de precio no se aplica en el nivel de base de datos - considere agregar la restricción CHECK")
	}
}

func (suite *ProductRepositoryTestSuite) TestProductPricing() {
	// Test various price scenarios
	priceTests := []struct {
		name  string
		price float64
		valid bool
	}{
		{"Zero price", 0.0, false}, // Debe ser inválido debido a la restricción CHECK
		{"Small price", 0.01, true},
		{"Normal price", 25.50, true},
		{"Large price", 999.99, true},
		{"Very large price", 9999.99, true},
	}

	for _, test := range priceTests {
		suite.T().Run(test.name, func(t *testing.T) {
			product := &models.Product{
				ProductID:   uuid.New(),
				Name:        "Producto de prueba de precio " + test.name,
				Description: "Probando precio: " + test.name,
				Price:       test.price,
				IsActive:    true,
			}

			err := suite.productRepo.Create(product)
			if test.valid {
				assert.NoError(t, err, "Debe aceptar el precio válido: %f", test.price)
			} else {
				assert.Error(t, err, "Debe rechazar el precio inválido: %f", test.price)
			}
		})
	}
}

func (suite *ProductRepositoryTestSuite) TestProductDescriptions() {
	// Test different descriptions
	descriptions := []string{
		"Balón de gas doméstico de 10kg",
		"Balón de gas industrial de 15kg",
		"Balón de gas medicinal certificado",
		"Regulador de gas estándar",
		"Manguera de gas de alta presión",
	}

	for i, description := range descriptions {
		product := &models.Product{
			ProductID:   uuid.New(),
			Name:        "Product " + string(rune(i+'A')),
			Description: description,
			Price:       25.50,
			IsActive:    true,
		}

		err := suite.productRepo.Create(product)
		assert.NoError(suite.T(), err, "Debe crear el producto con la descripción: %s", description)

		// Verify the description was stored correctly
		foundProduct, err := suite.productRepo.FindByID(product.ProductID.String())
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), description, foundProduct.Description)

		// Clean up for next iteration
		if i < len(descriptions)-1 {
			suite.db.Exec("DELETE FROM products WHERE product_id = ?", product.ProductID)
		}
	}
}

func (suite *ProductRepositoryTestSuite) TestActiveInactiveToggle() {
	// Create product
	product := &models.Product{
		ProductID:   uuid.New(),
		Name:        "Producto de prueba de activo/inactivo",
		Description: "Probando activo/inactivo",
		Price:       25.50,
		IsActive:    true,
	}

	err := suite.productRepo.Create(product)
	require.NoError(suite.T(), err)

	// Verify it appears in active products
	activeProducts, err := suite.productRepo.FindActive()
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), activeProducts, 1)

	// Deactivate product
	product.IsActive = false
	err = suite.productRepo.Update(product)
	assert.NoError(suite.T(), err)

	// Verify it doesn't appear in active products
	activeProducts, err = suite.productRepo.FindActive()
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), activeProducts, 0)

	// Verify it still appears in all products
	allProducts, err := suite.productRepo.FindAll()
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), allProducts, 1)
	assert.False(suite.T(), allProducts[0].IsActive)

	// Reactivate product
	product.IsActive = true
	err = suite.productRepo.Update(product)
	assert.NoError(suite.T(), err)

	// Verify it appears in active products again
	activeProducts, err = suite.productRepo.FindActive()
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), activeProducts, 1)
	assert.True(suite.T(), activeProducts[0].IsActive)
}

// TestProductRepositoryTestSuite runs the test suite
func TestProductRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(ProductRepositoryTestSuite))
}
