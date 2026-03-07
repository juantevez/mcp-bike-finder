// internal/infrastructure/vision/client.go
package vision

import (
	"context"
	"log"

	// ✅ Importar el paquete config para usar AWSConfig
	"github.com/juantevez/mcp-bike-finder/internal/config"
	"github.com/juantevez/mcp-bike-finder/internal/domain"
)

// ===========================================
// Client - Cliente de Visión por Computadora
// ===========================================

type Client struct {
	config Config
	mock   bool // true = modo mock para desarrollo
}

// ✅ Simplificar: usar directamente config.AWSConfig en lugar de vision.Config
type Config struct {
	Provider string
	Enabled  bool
	// Puedes mantener campos específicos de visión aquí si los necesitas
}

// ✅ NewClient ahora acepta config.AWSConfig directamente
func NewClient(ctx context.Context, cfg config.AWSConfig) (*Client, error) {
	return NewRekognitionClient(ctx, cfg)
}

// NewRekognitionClient crea un cliente para AWS Rekognition
func NewRekognitionClient(ctx context.Context, cfg config.AWSConfig) (*Client, error) {

	log.Println("👁️ [Vision] Inicializando cliente Rekognition (modo mock para desarrollo)")

	// Determinar si estamos en modo mock (desarrollo) o real (producción)
	mockMode := cfg.AccessKeyID == "" ||
		cfg.AccessKeyID == "tu_access_key" ||
		cfg.SecretAccessKey == "tu_secret_key"

	if mockMode {
		log.Println("👁️ [Vision] Modo MOCK: sin credenciales AWS reales")
	}

	return &Client{
		config: Config{
			Provider: "rekognition",
			Enabled:  !mockMode,
		},
		mock: mockMode,
	}, nil
}

// ===========================================
// Métodos Principales
// ===========================================

// AnalyzeBike analiza una imagen y extrae información específica de bicicletas
func (c *Client) AnalyzeBike(ctx context.Context, imageData []byte) (*domain.VisionResult, error) {

	if c.mock || !c.config.Enabled {
		return c.mockAnalyzeBike(imageData)
	}

	// 🚧 Aquí iría la llamada real a AWS Rekognition
	return c.mockAnalyzeBike(imageData)
}

// ValidateImage verifica si la imagen contiene una bicicleta
func (c *Client) ValidateImage(ctx context.Context, imageData []byte) (*domain.ImageValidation, error) {

	if c.mock || !c.config.Enabled {
		return c.mockValidateImage(imageData)
	}

	return c.mockValidateImage(imageData)
}

// ===========================================
// Implementaciones Mock (para desarrollo)
// ===========================================

func (c *Client) mockAnalyzeBike(imageData []byte) (*domain.VisionResult, error) {
	log.Println("👁️ [Vision-MOCK] Simulando análisis de bicicleta")

	return &domain.VisionResult{
		ColorDominante:    "azul",
		TipoBicicleta:     "mountain_bike",
		ObjetosDetectados: []string{"bicycle", "wheel", "frame", "handlebar", "suspension_fork"},
		Confianza:         0.94,
	}, nil
}

func (c *Client) mockValidateImage(imageData []byte) (*domain.ImageValidation, error) {
	log.Println("🔍 [Vision-MOCK] Validando imagen")

	return &domain.ImageValidation{
		EsBicicleta: true,
		Confianza:   0.96,
		Mensaje:     "Imagen válida: bicicleta de montaña detectada",
		Objetos:     []string{"bicycle", "wheel", "frame"},
	}, nil
}
