package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/LikhithMar14/BidZy/pkg/types"
)

type authStore struct {
	db *sql.DB
}

func NewAuthRepository(db *sql.DB) *authStore {
	return &authStore{db: db}
}

func (s *authStore) CreateUser(ctx context.Context, user *types.CreateUserRequest) (*types.User, error) {
	query := `INSERT INTO users (user_name, email, hashed_password) 
	          VALUES ($1, $2, $3) 
	          RETURNING id, user_name, email, created_at, updated_at;`

	var userResponse types.User

	err := s.db.QueryRowContext(ctx, query,
		user.UserName, user.Email, user.Password,
	).Scan(
		&userResponse.ID,
		&userResponse.UserName,
		&userResponse.Email,
		&userResponse.CreatedAt,
		&userResponse.UpdatedAt,
	)

	if err != nil {
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			return nil, errors.New("user already exists (username or email taken)")
		}
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return &userResponse, nil
}
