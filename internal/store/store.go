package store

import (
	"context"
	"database/sql"

	"github.com/LikhithMar14/BidZy/internal/store/auth"
	"github.com/LikhithMar14/BidZy/internal/store/bid"
	"github.com/LikhithMar14/BidZy/pkg/types"
)


type Store struct {
	Auth AuthRepository
	Bid BidRepositotry
}


type AuthRepository interface {
	CreateUser(ctx context.Context, user *types.CreateUserRequest, hashedPassword string) (*types.User, error)
	GetUserByEmailAndUserName(ctx context.Context, email , userName string) (*types.User, error)
}

type BidRepositotry interface {
	PlaceBid(ctx context.Context, bid *types.NewBidRequest) (*types.NewBidResponse, error)
	GetBidByID(ctx context.Context, id string) (*types.Bid, error)
	GetBidsByAuctionID(ctx context.Context, auctionID string) ([]*types.Bid, error)
	GetHighestBidForAuction(ctx context.Context, auctionID string) (*types.Bid, error)
	GetUserBids(ctx context.Context, userID string) ([]*types.Bid, error)
	GetBidCount(ctx context.Context, auctionID string) (int, error)
	GetBidHistoryWithPagination(ctx context.Context, auctionID string, limit, offset int) ([]*types.Bid, error)
}
func NewStorage(db *sql.DB) *Store {
	return &Store{
		Auth: auth.NewAuthRepository(db),
		Bid: bid.NewBidRepository(db),
	}
}