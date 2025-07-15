package config

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
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

// parseDuration parsea duraciones incluyendo días (ej: "7d")
func parseDuration(env string) (time.Duration, error) {
	log.Printf("🔍 DEBUG parseDuration: input='%s'", env)
	if strings.HasSuffix(env, "d") {
		daysStr := strings.TrimSuffix(env, "d")
		days, err := strconv.Atoi(daysStr)
		if err != nil {
			log.Printf("❌ DEBUG parseDuration: error parsing days: %v", err)
			return 0, err
		}
		result := time.Duration(days) * 24 * time.Hour
		log.Printf("✅ DEBUG parseDuration: %s -> %v (%d hours)", env, result, int(result.Hours()))
		return result, nil
	}
	result, err := time.ParseDuration(env)
	log.Printf("✅ DEBUG parseDuration: %s -> %v (error: %v)", env, result, err)
	return result, err
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

	// Configurar Viper para usar solo variables de entorno
	viper.AutomaticEnv()

	// Solo intentar leer archivo si existe y es readable
	if _, err := os.Stat("app.env"); err == nil {
		viper.SetConfigFile("app.env")
		if err := viper.ReadInConfig(); err != nil {
			// Log pero no fallar - las variables de entorno son suficientes
			log.Printf("Advertencia: No se pudo leer app.env: %v", err)
		}
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
			RefreshTokenExp: func() time.Duration {
				refreshTokenStr := viper.GetString("JWT_REFRESH_TOKEN_EXP")
				log.Printf("🔍 DEBUG JWT_REFRESH_TOKEN_EXP from viper: '%s'", refreshTokenStr)
				if duration, err := parseDuration(refreshTokenStr); err == nil {
					log.Printf("✅ DEBUG RefreshTokenExp set to: %v", duration)
					return duration
				} else {
					log.Printf("❌ DEBUG parseDuration failed, using fallback: %v", err)
					return 7 * 24 * time.Hour // fallback a 7 días
				}
			}(),
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
	viper.SetDefault("SERVER_HOST", "0.0.0.0")        // Cambio para Render.com
	viper.SetDefault("SERVER_PORT", getPortFromEnv()) // Usar PORT de Render si está disponible
	viper.SetDefault("SERVER_SHUTDOWN_TIMEOUT", "5s")
	viper.SetDefault("SERVER_READ_TIMEOUT", "5s")
	viper.SetDefault("SERVER_WRITE_TIMEOUT", "10s")
	viper.SetDefault("SERVER_IDLE_TIMEOUT", "120s")

	// Base de datos
	viper.SetDefault("DATABASE_URL", "") // Para Render.com
	viper.SetDefault("DB_HOST", "host.docker.internal")
	viper.SetDefault("DB_PORT", "5432")
	viper.SetDefault("DB_USER", "exactogas_user")
	viper.SetDefault("DB_PASSWORD", "exactogas_pass")
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

	// Establecer puerto por defecto si no se especifica
	port := parsedURL.Port()
	if port == "" {
		port = "5432" // Puerto por defecto de PostgreSQL
	}
	viper.Set("DB_PORT", port)
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
