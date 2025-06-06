package service

import (
	"context"

	"github.com/LikhithMar14/Bidzy/internal/models"
)


type Service struct {
	AuctionService AuctionServiceRepository
}

type AuctionServiceRepository interface {
	CreateAuction(ctx context.Context, auction *models.AuctionRequest) error
}

