package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// ===========================================
// Estructuras de Configuración
// ===========================================

type Config struct {
	MCP       MCPConfig
	DB        DBConfig
	AWS       AWSConfig
	OCR       OCRConfig
	Scraper   ScraperConfig
	Scheduler SchedulerConfig
}

type SchedulerConfig struct {
	IntervaloHoras int // cada cuántas horas correr las búsquedas automáticas
	LimiteBicis    int // máximo de bicis a procesar por ronda
}

type MCPConfig struct {
	Name      string
	Version   string
	Transport string // stdio, sse, http
	Port      string
}

type DBConfig struct {
	Host     string
	Port     string
	Name     string
	User     string
	Password string
	SSLMode  string
}

type AWSConfig struct {
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	S3Bucket        string
}

type OCRConfig struct {
	Provider        string // textract, google-vision
	TextractEnabled bool
	GoogleVisionKey string
}

type ScraperConfig struct {
	UserAgent    string
	DelayMs      int
	MaxRetries   int
	Marketplaces []string
}

// ===========================================
// Carga de Configuración
// ===========================================

func Load() (*Config, error) {
	// Cargar .env si existe (solo desarrollo local)
	_ = godotenv.Load()

	cfg := &Config{
		MCP: MCPConfig{
			Name:      getEnv("MCP_SERVER_NAME", "mcp-bike-finder"),
			Version:   getEnv("MCP_SERVER_VERSION", "1.0.0"),
			Transport: getEnv("MCP_TRANSPORT", "stdio"),
			Port:      getEnv("MCP_PORT", "8080"),
		},
		DB: DBConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			Name:     getEnv("DB_NAME", "bike_finder"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", ""),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		AWS: AWSConfig{
			Region:          getEnv("AWS_REGION", "us-east-1"),
			AccessKeyID:     getEnv("AWS_ACCESS_KEY_ID", ""),
			SecretAccessKey: getEnv("AWS_SECRET_ACCESS_KEY", ""),
			S3Bucket:        getEnv("S3_BUCKET_NAME", ""),
		},
		OCR: OCRConfig{
			Provider:        getEnv("OCR_PROVIDER", "textract"),
			TextractEnabled: getEnv("TEXTRACT_ENABLED", "true") == "true",
			GoogleVisionKey: getEnv("GOOGLE_VISION_API_KEY", ""),
		},
		Scraper: ScraperConfig{
			UserAgent:  getEnv("SCRAPER_USER_AGENT", "BikeFinderBot/1.0"),
			DelayMs:    1000,
			MaxRetries: 3,
			Marketplaces: []string{
				getEnv("MARKETPLACE_URLS", "https://mercadolibre.com"),
			},
		},
		Scheduler: SchedulerConfig{
			IntervaloHoras: getEnvInt("SCHEDULER_INTERVALO_HORAS", 6),
			LimiteBicis:    getEnvInt("SCHEDULER_LIMITE_BICIS", 100),
		},
	}

	// Validar configuración requerida
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	if c.AWS.AccessKeyID == "" || c.AWS.SecretAccessKey == "" {
		return fmt.Errorf("AWS_ACCESS_KEY_ID y AWS_SECRET_ACCESS_KEY son requeridos")
	}
	if c.AWS.S3Bucket == "" {
		return fmt.Errorf("S3_BUCKET_NAME es requerido")
	}
	if c.DB.Password == "" {
		return fmt.Errorf("DB_PASSWORD es requerido")
	}
	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var i int
		if _, err := fmt.Sscanf(value, "%d", &i); err == nil {
			return i
		}
	}
	return defaultValue
}
