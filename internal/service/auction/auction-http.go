package auction

import (
	"context"

	"github.com/LikhithMar14/BidZy/internal/store"
	"github.com/LikhithMar14/BidZy/pkg/types"
)


type AuctionHTTP struct {
	store store.AuctionRepository
}

func NewAuctionHTTP(store store.AuctionRepository) *AuctionHTTP {
	return &AuctionHTTP{
		store: store,
	}
}

	func (a *AuctionHTTP) CreateAuction(ctx context.Context, req *types.CreateAuctionRequest, categoryIDs []int,userID string) (*types.CreateAuctionResponse, error) {
	auction, err := a.store.CreateAuction(ctx, req, categoryIDs,userID)
	if err != nil {
		return nil, err
	}
	return &types.CreateAuctionResponse{
		ID: auction.ID,
		Title: auction.Title,
		Description: auction.Description,
		StartingPrice: auction.StartingPrice,
		CurrentPrice: auction.CurrentPrice,
		StartDateTime: auction.StartDateTime,
		EndDateTime: auction.EndDateTime,
		Status: auction.Status,
		Image: auction.Image,
		CreatedAt: auction.CreatedAt,
		UpdatedAt: auction.UpdatedAt,
		User: auction.User,
		CategoryIDs: auction.CategoryIDs,
		Increment: auction.Increment,
	}, nil

}