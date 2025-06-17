package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/LikhithMar14/BidZy/pkg/types"
)

type authStore struct {
	db *sql.DB
}

func NewAuthRepository(db *sql.DB) *authStore {
	return &authStore{db: db}
}

func (s *authStore) CreateUser(ctx context.Context, user *types.CreateUserRequest, hashedPassword string) (*types.User, error) {

	log.Println("Creating user", user.UserName, user.Email, hashedPassword)
	query := `INSERT INTO users (user_name, email, hashed_password) 
	          VALUES ($1, $2, $3) 
	          RETURNING id, user_name, email, hashed_password, created_at, updated_at;`

	var userResponse types.User

	err := s.db.QueryRowContext(ctx, query,
		user.UserName, user.Email, hashedPassword,
	).Scan(
		&userResponse.ID,
		&userResponse.UserName,
		&userResponse.Email,
		&userResponse.Password,
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
func (s *authStore) GetUserByEmailAndUserName(ctx context.Context, email, userName string) (*types.User, error) {
	if email == "" || userName == "" {
		return nil, errors.New("email and username are required")
	}

	query := `SELECT id, user_name, email, hashed_password, created_at, updated_at 
	          FROM users 
	          WHERE email = $1 OR user_name = $2;`

	var user types.User

	err := s.db.QueryRowContext(ctx, query, email, userName).Scan(
		&user.ID,
		&user.UserName,
		&user.Email,
		&user.Password,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		log.Println("User not found")
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

func (s *authStore) GetUserByID(ctx context.Context, id string) (*types.User, error) {
	query := `SELECT id, user_name, email,created_at, updated_at 
	          FROM users 
	          WHERE id = $1;`

	var user types.User

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.UserName,
		&user.Email,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		log.Println("User not found")
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}
