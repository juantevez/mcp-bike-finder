// internal/infrastructure/database/user_repository.go

package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/juantevez/mcp-bike-finder/internal/domain"
)

type UserRepository struct {
	pool       *pgxpool.Pool
	authSchema string
}

func NewUserRepository(pool *pgxpool.Pool, authSchema string) *UserRepository {
	return &UserRepository{
		pool:       pool,
		authSchema: authSchema,
	}
}

// ObtenerPorID - Para validar usuario en herramientas MCP
func (r *UserRepository) ObtenerPorID(ctx context.Context, id string) (*domain.User, error) {

	query := fmt.Sprintf(`
		SELECT id, email, email_verified, phone_number, phone_verified,
		       status, full_name, avatar_url, country_name, created_at, last_login_at
		FROM %s.users
		WHERE id = $1 AND status = 'ACTIVE'
	`, r.authSchema)

	var u domain.User
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&u.ID, &u.Email, &u.EmailVerified,
		&u.FullName, &u.AvatarURL, &u.Status,
		&u.CountryName, &u.CreatedAt, &u.LastLoginAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("usuario no encontrado o inactivo: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("error consultando usuario: %w", err)
	}

	return &u, nil
}

// ObtenerPorEmail - Para autenticación o lookup
func (r *UserRepository) ObtenerPorEmail(ctx context.Context, email string) (*domain.User, error) {

	query := fmt.Sprintf(`
		SELECT id, email, email_verified, status, full_name, avatar_url, created_at
		FROM %s.users
		WHERE LOWER(email) = LOWER($1) AND status = 'ACTIVE'
	`, r.authSchema)

	var u domain.User
	err := r.pool.QueryRow(ctx, query, email).Scan(
		&u.ID, &u.Email, &u.EmailVerified, &u.Status,
		&u.FullName, &u.AvatarURL, &u.CreatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &u, nil
}

// ContarBicicletasRegistradas - Para estadísticas de usuario
func (r *UserRepository) ContarBicicletasRegistradas(ctx context.Context, userID, schema string) (int, error) {

	query := fmt.Sprintf(`
		SELECT COUNT(*) 
		FROM %s.registered_bicycles
		WHERE user_id = $1 AND status = 'ACTIVE'
	`, schema)

	var count int
	err := r.pool.QueryRow(ctx, query, userID).Scan(&count)
	return count, err
}
