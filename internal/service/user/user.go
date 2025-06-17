package user

import (
	"context"
	"fmt"

	"github.com/LikhithMar14/BidZy/internal/store"
	"github.com/LikhithMar14/BidZy/pkg/types"
)

type UserService struct {
	store store.UserRepository
}

func NewUserService(store store.UserRepository) *UserService {
	return &UserService{store: store}
}

	func (s *UserService) GetUserByID(ctx context.Context) (*types.User, error) {
	userID, err := types.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	return s.store.GetUserByID(ctx, userID)
}

func (s *UserService) GetAuctionsByUserID(ctx context.Context) ([]*types.Auction, error) {
	userID, err := types.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	return s.store.GetAuctionsByUserID(ctx, userID)
}

func (s *UserService) GetBidsByUserID(ctx context.Context) ([]*types.Bid, error) {
	userID, err := types.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	return s.store.GetBidsByUserID(ctx, userID)
}

func (s *UserService) GetParticipatedAuctions(ctx context.Context) ([]*types.Auction, error) {
	userID, err := types.GetUserIDFromContext(ctx)
	fmt.Println("userID", userID)
	if err != nil {
		return nil, err
	}
	return s.store.GetParticipatedAuctions(ctx, userID)
}


func (s *UserService) GetOwnStats(ctx context.Context) (*types.UserStats, error) {
	userID, err := types.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	return s.store.GetUserStats(ctx, userID)
}


func (s *UserService) GetUserStatsByID(ctx context.Context) (*types.UserStats, error) {
	userID, err := types.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	return s.store.GetUserStats(ctx, userID)
}