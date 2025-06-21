package auction

import (
	"context"

	"github.com/LikhithMar14/BidZy/internal/store"
	"github.com/LikhithMar14/BidZy/pkg/types"
	"github.com/LikhithMar14/BidZy/pkg/utils"
)

type AuctionHTTP struct {
	store    store.AuctionRepository
	bidStore store.BidRepositotry
	uploader *utils.S3Uploader
}

func NewAuctionHTTP(store store.AuctionRepository, bidStore store.BidRepositotry, uploader *utils.S3Uploader) *AuctionHTTP {
	return &AuctionHTTP{
		store:    store,
		bidStore: bidStore,
		uploader: uploader,
	}
}

func (a *AuctionHTTP) CreateAuction(ctx context.Context, req *types.CreateAuctionRequest, categoryIDs []int, userID string) (*types.AuctionData, error) {
	// If imageKey is provided, generate S3 URL for the image
	if req.ImageKey != "" {
		req.Image = a.uploader.GenerateImageURL(req.ImageKey)
	}

	auction, err := a.store.CreateAuction(ctx, req, categoryIDs, userID)
	if err != nil {
		return nil, err
	}
	return auction, nil
}

func (a *AuctionHTTP) GetAuctionByID(ctx context.Context, auctionID string) (*types.AuctionData, error) {
	auction, err := a.store.GetAuctionByID(ctx, auctionID)
	if err != nil {
		return nil, err
	}
	return auction, nil
}

func (a *AuctionHTTP) GetAuctionsByUserID(ctx context.Context, userID string) ([]*types.AuctionData, error) {
	auctions, err := a.store.GetAuctionsByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return auctions, nil
}

func (a *AuctionHTTP) GetAllAuctions(ctx context.Context) ([]*types.AuctionData, error) {
	auctions, err := a.store.GetAllAuctions(ctx)
	if err != nil {
		return nil, err
	}
	return auctions, nil
}

func (a *AuctionHTTP) AddBid(ctx context.Context, bid *types.NewBidRequest) (*types.NewBidResponse, error) {
	bidResponse, err := a.bidStore.PlaceBid(ctx, bid)
	if err != nil {
		return nil, err
	}
	return bidResponse, nil
}

// UpdateClientCount updates the client count for an auction in the database
func (a *AuctionHTTP) UpdateClientCount(ctx context.Context, auctionID string, clientCount int) error {
	return a.store.UpdateAuctionClientCount(ctx, auctionID, clientCount)
}
