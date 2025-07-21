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
	HasAuctionEmailBeenSent(ctx context.Context, auctionID string) (bool, error)
	LogAuctionEmailSent(ctx context.Context, auctionID string) error
}

type MailService interface {
	SendAuctionEndedEmail(ctx context.Context, auctionID string) error
}

type AuctionScheduler struct {
	store       AuctionStore
	mailService MailService
	cron        *cron.Cron
}

func NewAuctionScheduler(store AuctionStore, mailService MailService) *AuctionScheduler {
	return &AuctionScheduler{
		store:       store,
		mailService: mailService,
		cron:        cron.New(cron.WithSeconds()),
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
		fmt.Println("==== CHECKING EMAILS TO SEND ====")
		fmt.Println("auctionIDs", auctionIDs)
		if err != nil {
			log.Println("[Scheduler] Failed to fetch recently ended auctions:", err)
			return
		}

		for _, auctionID := range auctionIDs {
			// 🔒 Check if email already sent (double-check for safety)
			sent, err := a.store.HasAuctionEmailBeenSent(ctx, auctionID)
			if err != nil {
				log.Printf("[Scheduler] ❌ Failed to check email log for auction %s: %v", auctionID, err)
				continue
			}
			if sent {
				log.Printf("[Scheduler] ✅ Email already sent for auction %s. Skipping.\n", auctionID)
				continue
			}

			log.Printf("[Scheduler] 📧 Sending auction ended email for auction %s...", auctionID)

			// ✅ Send email
			if err := a.mailService.SendAuctionEndedEmail(ctx, auctionID); err != nil {
				log.Printf("[Scheduler] ❌ Failed to send winner email for auction %s: %v", auctionID, err)
				continue
			}

			log.Printf("[Scheduler] ✅ Successfully sent emails for auction %s", auctionID)

			// ✏️ Log that email has been sent
			if err := a.store.LogAuctionEmailSent(ctx, auctionID); err != nil {
				log.Printf("[Scheduler] ⚠️ Email sent but failed to log for auction %s: %v", auctionID, err)
				// This is critical - if we can't log it, the email might be sent again
				// In production, you might want to implement a retry mechanism or alerting
			} else {
				log.Printf("[Scheduler] 🎯 Winner email sent and logged successfully for auction %s ✅", auctionID)
			}
		}
	})

	c.Start()
}

func (a *AuctionScheduler) Stop() {
	a.cron.Stop()
}
