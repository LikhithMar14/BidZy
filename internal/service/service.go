package service

import (
	"context"

	"github.com/LikhithMar14/BidZy/internal/service/auth"
	"github.com/LikhithMar14/BidZy/internal/store"
	"github.com/LikhithMar14/BidZy/pkg/types"
)

type AuthService interface {
	Register(ctx context.Context, req *types.CreateUserRequest) (*types.CreateUserResponse, error)
}

type Service struct {
	AuthService AuthService
}

func NewService(store *store.Store) *Service {
	return &Service{
		AuthService: auth.NewAuthService(store.Auth),
	}
}