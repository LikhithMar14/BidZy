package auction

import (
	"context"

	"github.com/LikhithMar14/BidZy/internal/store"
	"github.com/LikhithMar14/BidZy/pkg/types"
)

type AuctionHTTP struct {
	store    store.AuctionRepository
	bidStore store.BidRepositotry
}

func NewAuctionHTTP(store store.AuctionRepository, bidStore store.BidRepositotry) *AuctionHTTP {
	return &AuctionHTTP{
		store:    store,
		bidStore: bidStore,
	}
}

func (a *AuctionHTTP) CreateAuction(ctx context.Context, req *types.CreateAuctionRequest, categoryIDs []int, userID string) (*types.AuctionData, error) {
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
