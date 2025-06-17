package bid

import (
	"context"

	"github.com/LikhithMar14/BidZy/internal/store"
	"github.com/LikhithMar14/BidZy/pkg/types"
)

type BidService struct {
	store store.BidRepositotry
}

func NewBidService(store store.BidRepositotry) *BidService {
	return &BidService{store: store}
}

func (s *BidService) GetBidTimelineByAuctionID(ctx context.Context, auctionID string) ([]*types.BidTimelineEntry, error) {
	return s.store.GetBidTimelineByAuctionID(ctx, auctionID)
}