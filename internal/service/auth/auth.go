package auth

import (
	"context"

	"github.com/LikhithMar14/BidZy/internal/store"
	"github.com/LikhithMar14/BidZy/pkg/types"
)


type AuthService struct {
	store store.AuthRepository
}

func NewAuthService(store store.AuthRepository) *AuthService {
	return &AuthService{store: store}
}

func (s *AuthService) Register(ctx context.Context, req *types.CreateUserRequest) (*types.CreateUserResponse, error) {
	//validate the request will do later
	
	user, err := s.store.CreateUser(ctx, req)

	var userResponse types.CreateUserResponse

	if err != nil {
		return nil, err
	}

	userResponse.User = *user
	userResponse.Token = "e23yijf90"

	return &userResponse, nil
}

