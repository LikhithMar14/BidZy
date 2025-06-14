package scheduler

import (
	"context"
	"log"

	"github.com/robfig/cron/v3"
)

type AuctionStore interface {
	MarkAuctionsActive(ctx context.Context) error
	MarkAuctionsEnded(ctx context.Context) error
}

type AuctionScheduler struct {
	store AuctionStore
}

func NewAuctionScheduler(store AuctionStore) *AuctionScheduler {
	return &AuctionScheduler{store: store}
}

func (a *AuctionScheduler) Start() {
	// Enable seconds in cron (default is only per minute)
	c := cron.New(cron.WithSeconds())

	// Run every 30 seconds - FIXED cron expression
	c.AddFunc("*/30 * * * * *", func() {
		log.Println("[Scheduler] Running auction status update task...")
		ctx := context.Background()

		// 1. First activate auctions that should be active
		if err := a.store.MarkAuctionsActive(ctx); err != nil {
			log.Println("[Scheduler] Failed to mark auctions as active:", err)
		}

		// 2. Then end expired auctions
		if err := a.store.MarkAuctionsEnded(ctx); err != nil {
			log.Println("[Scheduler] Failed to mark auctions as ended:", err)
		}
	})

	c.Start()
}
