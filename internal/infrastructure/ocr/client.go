// internal/infrastructure/ocr/client.go
package ocr

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/juantevez/mcp-bike-finder/internal/config"
)

// ===========================================
// Client - Cliente OCR para extracción de texto
// ===========================================

type Client struct {
	config Config
	mock   bool // true = modo mock para desarrollo
}

// ✅ Simplificar: usar directamente config.AWSConfig en lugar de ocr.Config
type Config struct {
	Provider string
	Enabled  bool
	// Puedes mantener campos específicos de OCR aquí si los necesitas
}

// ✅ NewClient ahora acepta config.AWSConfig directamente
func NewClient(ctx context.Context, cfg config.AWSConfig) (*Client, error) {
	return NewTextractClient(ctx, cfg)
}

// NewTextractClient crea un cliente para AWS Textract
func NewTextractClient(ctx context.Context, cfg config.AWSConfig) (*Client, error) {

	log.Println("📝 [OCR] Inicializando cliente Textract (modo mock para desarrollo)")

	// Determinar si estamos en modo mock (desarrollo) o real (producción)
	mockMode := cfg.AccessKeyID == "" ||
		cfg.AccessKeyID == "tu_access_key" ||
		cfg.SecretAccessKey == "tu_secret_key"

	if mockMode {
		log.Println("📝 [OCR] Modo MOCK: sin credenciales AWS reales")
	}

	return &Client{
		config: Config{
			Provider: "textract",
			Enabled:  !mockMode,
			// ✅ No agregamos AWSRegion/AccessKeyID/SecretKey porque NO existen en ocr.Config
			// Si los necesitas para producción futura, agrégalos al struct Config arriba
		},
		mock: mockMode,
	}, nil
}

// ===========================================
// Método Principal: DetectText
// ===========================================

// DetectText extrae texto de una imagen (PNG/JPG)
func (c *Client) DetectText(ctx context.Context, imageData []byte) (string, error) {

	if c.mock || !c.config.Enabled {
		return c.mockDetectText(imageData)
	}

	// 🚧 Aquí iría la llamada real a AWS Textract
	// return c.detectTextWithTextract(ctx, imageData)

	return c.mockDetectText(imageData)
}

// ===========================================
// Implementación Mock (para desarrollo)
// ===========================================

func (c *Client) mockDetectText(imageData []byte) (string, error) {
	log.Println("📝 [OCR-MOCK] Simulando extracción de texto de imagen")

	// Simular texto que podría extraerse de una foto de bicicleta
	// En producción, esto vendría de AWS Textract / Google Vision
	/*mockText := `
		TREK MARLIN 7
		Model Year: 2022
		Color: Azul Metallic
		Frame Size: M (17.5")
		
		Components:
		- Fork: Suntour XCE 28, 100mm travel
		- Drivetrain: Shimano Altus 2x8 speed
		- Brakes: Tektro HD-M275 hydraulic disc
		- Wheels: Bontrager Connector disc, 29"
		- Saddle: Bontrager Evoke
		- Handlebar: Bontrager Alloy, 31.8mm
		
		Serial: TRK2022M789456
		Purchased: 2022-03-15
	`*/
	mockText:= fmt.Sprintf("info: ", imageData)

	return strings.TrimSpace(mockText), nil
}

// ===========================================
// Implementación Real (Ejemplo AWS Textract)
// ===========================================

/*
// En producción, descomentar y configurar:

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/textract"
	"github.com/aws/aws-sdk-go-v2/service/textract/types"
)

func (c *Client) detectTextWithTextract(ctx context.Context, imageData []byte) (string, error) {

	// Cargar configuración AWS
	awsCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(c.config.AWSRegion))
	if err != nil {
		return "", fmt.Errorf("error cargando config AWS: %w", err)
	}

	// Crear cliente Textract
	client := textract.NewFromConfig(awsCfg)

	// Preparar request
	input := &textract.DetectDocumentTextInput{
		Document: &types.Document{
			Bytes: imageData,
		},
	}

	// Ejecutar
	result, err := client.DetectDocumentText(ctx, input)
	if err != nil {
		return "", fmt.Errorf("error en Textract: %w", err)
	}

	// Extraer texto de los bloques
	var texto strings.Builder
	for _, block := range result.Blocks {
		if block.BlockType == types.BlockTypeLine {
			texto.WriteString(*block.Text + " ")
		}
	}

	return strings.TrimSpace(texto.String()), nil
}
*/
