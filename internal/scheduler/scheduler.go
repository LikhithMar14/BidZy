package scheduler

import (
	"context"
	"fmt"
	"log"

	"github.com/robfig/cron/v3"
)

type AuctionStore interface {
	MarkAuctionsActive(ctx context.Context) error
	MarkAuctionsEnded(ctx context.Context) error
	GetRecentlyEndedAuctionIDs(ctx context.Context) ([]string, error)
}

type MailService interface {
	SendAuctionEndedEmail(ctx context.Context, auctionID string) error
}

type AuctionScheduler struct {
	store       AuctionStore
	mailService MailService
}

func NewAuctionScheduler(store AuctionStore, mailService MailService) *AuctionScheduler {
	return &AuctionScheduler{
		store:       store,
		mailService: mailService,
	}
}

func (a *AuctionScheduler) Start() {
	c := cron.New(cron.WithSeconds())

	// Runs every 30 seconds
	c.AddFunc("*/30 * * * * *", func() {
		log.Println("[Scheduler] Running auction status + winner mail task...")
		ctx := context.Background()

		// 1. Activate auctions that should start
		if err := a.store.MarkAuctionsActive(ctx); err != nil {
			log.Println("[Scheduler] Failed to activate auctions:", err)
		}

		// 2. End auctions whose time is over
		if err := a.store.MarkAuctionsEnded(ctx); err != nil {
			log.Println("[Scheduler] Failed to end auctions:", err)
		}

		// 3. Send emails to winners of just-ended auctions
		auctionIDs, err := a.store.GetRecentlyEndedAuctionIDs(ctx)
		fmt.Println("==== CHEKCING EMAILS TO SEND ====")
		fmt.Println("auctionIDs", auctionIDs)
		if err != nil {
			log.Println("[Scheduler] Failed to fetch recently ended auctions:", err)
			return
		}

		for _, auctionID := range auctionIDs {
			fmt.Println("auctionID", auctionID)
			if err := a.mailService.SendAuctionEndedEmail(ctx, auctionID); err != nil {
				log.Printf("[Scheduler] Failed to send winner email for auction %s: %v", auctionID, err)
			} else {
				log.Printf("[Scheduler] Winner email sent for auction %s ✅", auctionID)
			}
		}
	})

	c.Start()
}
