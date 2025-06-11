package store

import (
	"context"
	"database/sql"

	"github.com/LikhithMar14/BidZy/internal/store/auth"
	"github.com/LikhithMar14/BidZy/pkg/types"
)


type Store struct {
	Auth AuthRepository
}


type AuthRepository interface {
	CreateUser(ctx context.Context, user *types.CreateUserRequest) (*types.User, error)
}

func NewStorage(db *sql.DB) *Store {
	return &Store{
		Auth: auth.NewAuthRepository(db),
	}
}