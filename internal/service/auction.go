package service

import (
	"context"

	"github.com/LikhithMar14/Bidzy/internal/models"
	"github.com/LikhithMar14/Bidzy/internal/store"
)


type AuctionService struct {
	auctionRepo store.AuctionRepository
}

func NewAuctionService(auctionRepo store.AuctionRepository) *AuctionService {
	return &AuctionService{
		auctionRepo: auctionRepo,
	}
}

func (s *AuctionService) CreateAuction(ctx context.Context, auction *models.AuctionRequest) error {
	return s.auctionRepo.CreateAuction(ctx, auction)
}	