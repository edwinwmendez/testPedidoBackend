package config

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// Config contiene toda la configuración de la aplicación
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	JWT      JWTConfig
	Firebase FirebaseConfig
	App      AppConfig
}

// ServerConfig contiene la configuración del servidor HTTP
type ServerConfig struct {
	Host            string
	Port            string
	ShutdownTimeout time.Duration
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
}

// DatabaseConfig contiene la configuración de la base de datos
type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// JWTConfig contiene la configuración para JWT
type JWTConfig struct {
	Secret          string
	AccessTokenExp  time.Duration
	RefreshTokenExp time.Duration
}

// FirebaseConfig contiene la configuración para Firebase
type FirebaseConfig struct {
	ProjectID       string
	CredentialsFile string
}

// AppConfig contiene la configuración específica de la aplicación
type AppConfig struct {
	BusinessHoursStart time.Duration // Hora de inicio del horario de atención (en horas desde medianoche)
	BusinessHoursEnd   time.Duration // Hora de fin del horario de atención (en horas desde medianoche)
	TimeZone           string        // Zona horaria para el horario de atención
}

// LoadConfig carga la configuración desde variables de entorno o archivo .env
func LoadConfig() (*Config, error) {
	// Cargar .env si existe (solo en desarrollo local)
	envFile := "app.env"
	if _, err := os.Stat(envFile); err == nil {
		if err := godotenv.Load(envFile); err != nil {
			log.Printf("Error al cargar %s: %v", envFile, err)
		}
	}

	// Configurar Viper
	viper.SetConfigName("app")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("../")
	viper.AutomaticEnv()

	// Cargar configuración
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error al leer el archivo de configuración: %w", err)
		}
		log.Println("No se encontró archivo de configuración, usando variables de entorno")
	}

	// Establecer valores por defecto
	setDefaults()

	// Parsear DATABASE_URL si está disponible (Render.com/producción)
	if databaseURL := viper.GetString("DATABASE_URL"); databaseURL != "" {
		if err := parseAndSetDatabaseURL(databaseURL); err != nil {
			log.Printf("Error al parsear DATABASE_URL: %v, usando configuración por defecto", err)
		}
	}

	// Crear y poblar la estructura de configuración
	cfg := &Config{
		Server: ServerConfig{
			Host:            viper.GetString("SERVER_HOST"),
			Port:            viper.GetString("SERVER_PORT"),
			ShutdownTimeout: viper.GetDuration("SERVER_SHUTDOWN_TIMEOUT"),
			ReadTimeout:     viper.GetDuration("SERVER_READ_TIMEOUT"),
			WriteTimeout:    viper.GetDuration("SERVER_WRITE_TIMEOUT"),
			IdleTimeout:     viper.GetDuration("SERVER_IDLE_TIMEOUT"),
		},
		Database: DatabaseConfig{
			Host:     viper.GetString("DB_HOST"),
			Port:     viper.GetString("DB_PORT"),
			User:     viper.GetString("DB_USER"),
			Password: viper.GetString("DB_PASSWORD"),
			DBName:   viper.GetString("DB_NAME"),
			SSLMode:  viper.GetString("DB_SSLMODE"),
		},
		JWT: JWTConfig{
			Secret:          viper.GetString("JWT_SECRET"),
			AccessTokenExp:  viper.GetDuration("JWT_ACCESS_TOKEN_EXP"),
			RefreshTokenExp: viper.GetDuration("JWT_REFRESH_TOKEN_EXP"),
		},
		Firebase: FirebaseConfig{
			ProjectID:       viper.GetString("FIREBASE_PROJECT_ID"),
			CredentialsFile: viper.GetString("FIREBASE_CREDENTIALS_FILE"),
		},
		App: AppConfig{
			BusinessHoursStart: viper.GetDuration("APP_BUSINESS_HOURS_START"),
			BusinessHoursEnd:   viper.GetDuration("APP_BUSINESS_HOURS_END"),
			TimeZone:           viper.GetString("APP_TIMEZONE"),
		},
	}

	return cfg, nil
}

// setDefaults establece valores por defecto para la configuración
func setDefaults() {
	// Servidor
	viper.SetDefault("SERVER_HOST", "0.0.0.0") // Cambio para Render.com
	viper.SetDefault("SERVER_PORT", getPortFromEnv()) // Usar PORT de Render si está disponible
	viper.SetDefault("SERVER_SHUTDOWN_TIMEOUT", "5s")
	viper.SetDefault("SERVER_READ_TIMEOUT", "5s")
	viper.SetDefault("SERVER_WRITE_TIMEOUT", "10s")
	viper.SetDefault("SERVER_IDLE_TIMEOUT", "120s")

	// Base de datos
	viper.SetDefault("DATABASE_URL", "") // Para Render.com
	viper.SetDefault("DB_HOST", "localhost")
	viper.SetDefault("DB_PORT", "5432")
	viper.SetDefault("DB_USER", "postgres")
	viper.SetDefault("DB_PASSWORD", "postgres")
	viper.SetDefault("DB_NAME", "exactogas")
	viper.SetDefault("DB_SSLMODE", "disable")

	// JWT
	viper.SetDefault("JWT_SECRET", "your-256-bit-secret")
	viper.SetDefault("JWT_ACCESS_TOKEN_EXP", "15m")
	viper.SetDefault("JWT_REFRESH_TOKEN_EXP", "7d")

	// Firebase
	viper.SetDefault("FIREBASE_PROJECT_ID", "exactogas-app")
	viper.SetDefault("FIREBASE_CREDENTIALS_FILE", filepath.Join("config", "firebase-credentials.json"))

	// Aplicación
	viper.SetDefault("APP_BUSINESS_HOURS_START", "6h") // 6:00 AM
	viper.SetDefault("APP_BUSINESS_HOURS_END", "20h")  // 8:00 PM
	viper.SetDefault("APP_TIMEZONE", "America/Lima")   // Zona horaria de Perú
}

// parseAndSetDatabaseURL parsea una URL de base de datos completa y establece las variables individuales
func parseAndSetDatabaseURL(databaseURL string) error {
	parsedURL, err := url.Parse(databaseURL)
	if err != nil {
		return fmt.Errorf("error al parsear DATABASE_URL: %w", err)
	}

	// Extraer componentes de la URL
	viper.Set("DB_HOST", parsedURL.Hostname())
	viper.Set("DB_PORT", parsedURL.Port())
	viper.Set("DB_USER", parsedURL.User.Username())
	
	if password, ok := parsedURL.User.Password(); ok {
		viper.Set("DB_PASSWORD", password)
	}
	
	// El nombre de la base de datos está en el path, removiendo el '/' inicial
	if len(parsedURL.Path) > 1 {
		viper.Set("DB_NAME", strings.TrimPrefix(parsedURL.Path, "/"))
	}

	// En producción (Render), usar SSL
	if parsedURL.Hostname() != "localhost" && parsedURL.Hostname() != "127.0.0.1" {
		viper.Set("DB_SSLMODE", "require")
	} else {
		viper.Set("DB_SSLMODE", "disable")
	}

	return nil
}

// getPortFromEnv obtiene el puerto desde la variable de entorno PORT (Render) o usa 8080 por defecto
func getPortFromEnv() string {
	if port := os.Getenv("PORT"); port != "" {
		return port
	}
	return "8080"
}

// GetDSN retorna el DSN para la conexión a la base de datos
func (c *DatabaseConfig) GetDSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode,
	)
}
