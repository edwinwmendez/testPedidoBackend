package database

import (
	"fmt"
	"log"
	"os"
	"time"

	"backend/config"
	"backend/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB es la instancia global de la base de datos
var DB *gorm.DB

// Connect establece la conexión con la base de datos PostgreSQL
func Connect(cfg *config.Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=America/Lima",
		cfg.Database.Host,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.DBName,
		cfg.Database.Port,
	)

	// Configuración del logger de GORM
	gormLogger := logger.New(
		log.New(log.Writer(), "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logger.Error,
			IgnoreRecordNotFoundError: true,
			Colorful:                  true,
		},
	)

	// Intentar conectar a la base de datos con reintentos
	var db *gorm.DB
	var err error
	maxRetries := 5
	for i := 0; i < maxRetries; i++ {
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
			Logger: gormLogger,
		})
		if err == nil {
			break
		}
		log.Printf("Error al conectar a la base de datos (intento %d/%d): %v", i+1, maxRetries, err)
		time.Sleep(time.Second * 5)
	}

	if err != nil {
		return nil, fmt.Errorf("error al conectar a la base de datos: %w", err)
	}

	// Configurar pool de conexiones
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("error al obtener la conexión SQL: %w", err)
	}

	// Configurar el pool de conexiones
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// Guardar la instancia global
	DB = db

	// Ejecutar migraciones automáticamente en producción (Render)
	if os.Getenv("RENDER") == "true" {
		log.Println("Entorno de producción detectado - ejecutando migraciones automáticas...")
		if err := MigrateSchema(db); err != nil {
			log.Printf("Advertencia: Error en migración automática: %v", err)
			// No fallar completamente - la app puede funcionar con esquema existente
		}
	}

	log.Println("Conexión a la base de datos establecida exitosamente")
	return db, nil
}

// GetDB retorna la instancia global de la base de datos
func GetDB() *gorm.DB {
	return DB
}

// MigrateSchema ejecuta las migraciones para crear/actualizar el esquema de la base de datos
func MigrateSchema(db *gorm.DB) error {
	log.Println("Iniciando migración del esquema de la base de datos...")

	// Crear los tipos ENUM requeridos
	// Nota: En PostgreSQL, esto requiere una migración SQL específica.
	// Aquí usamos el enfoque básico de GORM, pero para producción debería usarse migración SQL.

	// Deshabilitar temporalmente la creación automática de FK para evitar problemas
	db.DisableForeignKeyConstraintWhenMigrating = true
	
	// Migrar tablas base primero (sin relaciones)
	err := db.AutoMigrate(&models.User{}, &models.Product{})
	if err != nil {
		return fmt.Errorf("error al migrar tablas base: %w", err)
	}
	
	// Luego migrar tablas con relaciones
	err = db.AutoMigrate(&models.Order{}, &models.OrderItem{})
	if err != nil {
		return fmt.Errorf("error al migrar tablas con relaciones: %w", err)
	}
	
	// Habilitar nuevamente las FK para operaciones futuras
	db.DisableForeignKeyConstraintWhenMigrating = false

	log.Println("Migración del esquema completada exitosamente")
	return nil
}

// SeedInitialData carga datos iniciales en la base de datos (para desarrollo/demo)
func SeedInitialData(db *gorm.DB) error {
	// Verificar si ya existen productos
	var count int64
	db.Model(&models.Product{}).Count(&count)
	if count > 0 {
		log.Println("Datos iniciales ya existen, omitiendo carga...")
		return nil
	}

	log.Println("Cargando datos iniciales...")

	// Productos de ejemplo
	products := []models.Product{
		{
			Name:        "Balón de Gas 10kg",
			Description: "Balón de gas doméstico de 10 kilogramos, para cocina y uso general",
			Price:       50.00,
			IsActive:    true,
		},
		{
			Name:        "Balón de Gas 5kg",
			Description: "Balón de gas doméstico de 5 kilogramos, ideal para uso ocasional o espacios reducidos",
			Price:       30.00,
			IsActive:    true,
		},
		{
			Name:        "Balón de Gas 15kg",
			Description: "Balón de gas doméstico de 15 kilogramos, para uso intensivo o comercios pequeños",
			Price:       70.00,
			IsActive:    true,
		},
	}

	// Insertar productos
	if err := db.Create(&products).Error; err != nil {
		return fmt.Errorf("error al insertar productos iniciales: %w", err)
	}

	// Aquí se pueden agregar más datos iniciales si es necesario
	// Por ejemplo, un usuario administrador por defecto

	log.Println("Datos iniciales cargados exitosamente")
	return nil
}
