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
	c := cron.New()

	c.AddFunc("@every 1m", func() {
		log.Println("[Scheduler] Running auction status update task...")
		ctx := context.Background()

		if err := a.store.MarkAuctionsActive(ctx); err != nil {
			log.Println("[Scheduler] Failed to mark auctions as active:", err)
		}
		if err := a.store.MarkAuctionsEnded(ctx); err != nil {
			log.Println("[Scheduler] Failed to mark auctions as ended:", err)
		}
	})

	c.Start()
}
