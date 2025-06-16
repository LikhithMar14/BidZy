package store

import (
	"context"
	"database/sql"

	"github.com/LikhithMar14/BidZy/internal/store/auction"
	"github.com/LikhithMar14/BidZy/internal/store/auth"
	"github.com/LikhithMar14/BidZy/internal/store/bid"
	"github.com/LikhithMar14/BidZy/internal/store/category"
	"github.com/LikhithMar14/BidZy/pkg/types"
)

type Store struct {
	Auth     AuthRepository
	Bid      BidRepositotry
	Category CategoryRepository
	Auction  AuctionRepository
}

type AuthRepository interface {
	CreateUser(ctx context.Context, user *types.CreateUserRequest, hashedPassword string) (*types.User, error)
	GetUserByEmailAndUserName(ctx context.Context, email, userName string) (*types.User, error)
	GetUserByID(ctx context.Context, id string) (*types.User, error)
}

type CategoryRepository interface {
	GetAllCategories(ctx context.Context) ([]*types.Category, error)
}

type AuctionRepository interface {
	CreateAuction(ctx context.Context, auction *types.CreateAuctionRequest, categoryIDs []int, userID string) (*types.AuctionData, error)
	MarkAuctionsActive(ctx context.Context) error
	MarkAuctionsEnded(ctx context.Context) error
	GetAllAuctions(ctx context.Context) ([]*types.AuctionData, error)
	GetAuctionByID(ctx context.Context, auctionID string) (*types.AuctionData, error)
	GetAuctionsByUserID(ctx context.Context, userID string) ([]*types.AuctionData, error)
	AddCategoryToAuction(ctx context.Context, auctionID, categoryID string) error
	RemoveCategoryFromAuction(ctx context.Context, auctionID, categoryID string) error
	RemoveAllCategoriesFromAuction(ctx context.Context, auctionID string) error
	GetRecentlyEndedAuctionIDs(ctx context.Context) ([]string, error)
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
		Auth:     auth.NewAuthRepository(db),
		Bid:      bid.NewBidRepository(db),
		Category: category.NewCategoryRepository(db),
		Auction:  auction.NewAuctionRepository(db),
	}
}
