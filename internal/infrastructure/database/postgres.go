package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/juantevez/mcp-bike-finder/internal/config"
)

// ===========================================
// PostgresDB - Wrapper para pgxpool
// ===========================================

type PostgresDB struct {
	pool       *pgxpool.Pool
	schema     string // "bike" para tablas de negocio
	authSchema string // "auth" para usuarios
}

// NewPostgresDB crea y verifica la conexión a PostgreSQL
func NewPostgresDB(ctx context.Context, cfg config.DBConfig) (*PostgresDB, error) {

	// Connection string compatible con pgx/v5
	connStr := fmt.Sprintf(
		"host=%s port=%s dbname=%s user=%s password=%s sslmode=%s application_name=mcp-bike-finder",
		cfg.Host, cfg.Port, cfg.Name, cfg.User, cfg.Password, cfg.SSLMode,
	)

	// Configurar pool de conexiones
	poolConfig, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf("error parseando conexión: %w", err)
	}

	// Optimizaciones para MCP (baja latencia, pocas conexiones simultáneas)
	poolConfig.MaxConns = 10
	poolConfig.MinConns = 2
	poolConfig.MaxConnLifetime = 30 * time.Minute
	poolConfig.MaxConnIdleTime = 10 * time.Minute
	poolConfig.HealthCheckPeriod = 1 * time.Minute

	// Crear pool
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("error creando pool: %w", err)
	}

	// Verificar conexión con retry
	if err := verifyConnection(ctx, pool, 3); err != nil {
		pool.Close()
		return nil, fmt.Errorf("error conectando a PostgreSQL: %w", err)
	}

	log.Printf("✅ Conexión a PostgreSQL establecida: %s@%s:%s/%s",
		cfg.User, cfg.Host, cfg.Port, cfg.Name)

	return &PostgresDB{
		pool:       pool,
		schema:     "bike",
		authSchema: "auth",
	}, nil
}

// verifyConnection intenta conectar con reintentos
func verifyConnection(ctx context.Context, pool *pgxpool.Pool, maxRetries int) error {
	for i := 0; i < maxRetries; i++ {
		if err := pool.Ping(ctx); err == nil {
			return nil
		}
		if i < maxRetries-1 {
			log.Printf("⏳ Reintentando conexión (%d/%d)...", i+1, maxRetries)
			time.Sleep(2 * time.Second)
		}
	}
	return pool.Ping(ctx)
}

// Close cierra el pool de conexiones
func (db *PostgresDB) Close() {
	if db.pool != nil {
		db.pool.Close()
		log.Println("🔌 Conexión a PostgreSQL cerrada")
	}
}

// Pool expone el pool para queries directas si es necesario
func (db *PostgresDB) Pool() *pgxpool.Pool {
	return db.pool
}

// Schemas para construir queries dinámicas
func (db *PostgresDB) BikeSchema() string { return db.schema }
func (db *PostgresDB) AuthSchema() string { return db.authSchema }
