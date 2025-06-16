package service

import (
	"context"

	auction "github.com/LikhithMar14/BidZy/internal/service/auction"
	"github.com/LikhithMar14/BidZy/internal/service/auth"
	"github.com/LikhithMar14/BidZy/internal/service/category"
	"github.com/LikhithMar14/BidZy/internal/store"
	"github.com/LikhithMar14/BidZy/pkg/types"
	"golang.org/x/oauth2"
)

type AuthService interface {
	Register(ctx context.Context, req *types.CreateUserRequest) (*types.CreateUserResponse, error)
	Login(ctx context.Context, req *types.LoginRequest) (*types.LoginResponse, error)
	GetUserInfoFromGoogle(ctx context.Context, code string) (map[string]interface{}, error)
	GetGoogleLoginURL(state string) string
}

type CategoryService interface {
	GetAllCategories(ctx context.Context) ([]*types.Category, error)
}

type AuctionService interface {
	CreateAuction(ctx context.Context, req *types.CreateAuctionRequest, categoryIDs []int, userID string) (*types.AuctionData, error)
	GetAllAuctions(ctx context.Context) ([]*types.AuctionData, error)
	GetAuctionByID(ctx context.Context, auctionID string) (*types.AuctionData, error)
	GetAuctionsByUserID(ctx context.Context, userID string) ([]*types.AuctionData, error)
	AddBid(ctx context.Context, req *types.NewBidRequest) (*types.NewBidResponse, error)
}

type Service struct {
	AuthService     AuthService
	CategoryService CategoryService
	AuctionService  AuctionService
}

func NewService(store *store.Store, jwtSecret string, googleOauthClient *oauth2.Config) *Service {
	return &Service{
		AuthService: auth.NewAuthService(store.Auth, jwtSecret, googleOauthClient),

		CategoryService: category.NewCategoryService(store.Category),
		AuctionService:  auction.NewAuctionHTTP(store.Auction, store.Bid),
	}
}
