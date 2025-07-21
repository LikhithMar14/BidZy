package mail

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/LikhithMar14/BidZy/internal/store"
	"github.com/LikhithMar14/BidZy/pkg/types"
	"gopkg.in/gomail.v2"
)

type EmailMessage struct {
	To      string
	Subject string
	Body    string
	IsHTML  bool
}

type SMTPConfig struct {
	Host   string
	Port   int
	User   string
	Pass   string
	From   string
	Secure string
}

func Load() (*SMTPConfig, error) {
	port, err := strconv.Atoi(os.Getenv("SMTP_PORT"))
	if err != nil {
		return nil, fmt.Errorf("invalid port: %w", err)
	}

	return &SMTPConfig{
		Host:   os.Getenv("SMTP_HOST"),
		Port:   port,
		User:   os.Getenv("SMTP_USER"),
		Pass:   os.Getenv("SMTP_PASS"),
		From:   os.Getenv("SMTP_FROM"),
		Secure: os.Getenv("SMTP_SECURE"),
	}, nil
}

func SendEmail(cfg *SMTPConfig, msg EmailMessage) error {
	m := gomail.NewMessage()
	m.SetHeader("From", cfg.From)
	m.SetHeader("To", msg.To)
	m.SetHeader("Subject", msg.Subject)

	contentType := "text/plain"
	if msg.IsHTML {
		contentType = "text/html"
	}
	m.SetBody(contentType, msg.Body)

	dialer := gomail.NewDialer(cfg.Host, cfg.Port, cfg.User, cfg.Pass)
	dialer.SSL = cfg.Secure == "true"

	if err := dialer.DialAndSend(m); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}
	return nil
}

func NewMailer(cfg *SMTPConfig) *MailService {
	return &MailService{
		cfg: cfg,
	}
}

func NewMailService(cfg *SMTPConfig, auctionStore store.AuctionRepository, authStore store.AuthRepository, bidStore store.BidRepositotry) *MailService {
	return &MailService{
		cfg:          cfg,
		auctionStore: auctionStore,
		authStore:    authStore,
		bidStore:     bidStore,
	}
}

type MailService struct {
	cfg          *SMTPConfig
	auctionStore store.AuctionRepository
	authStore    store.AuthRepository
	bidStore     store.BidRepositotry
}

func (s *MailService) SendEmail(ctx context.Context, msg EmailMessage) error {
	return SendEmail(s.cfg, msg)
}

func (s *MailService) SendAuctionEndedEmail(ctx context.Context, auctionID string) error {
	fmt.Printf("[Mail] 📧 Starting email send for auction %s\n", auctionID)

	// Step 1: Fetch auction
	auction, err := s.auctionStore.GetAuctionByID(ctx, auctionID)
	if err != nil || auction == nil {
		return fmt.Errorf("auction not found: %w", err)
	}

	if auction.Status != "ENDED" {
		fmt.Printf("[Mail] ⚠️ Auction %s status is %s, not ENDED. Skipping email send.\n", auctionID, auction.Status)
		return nil // Only send if the auction is marked ENDED
	}

	// Step 2: Get highest bid
	highestBid, err := s.bidStore.GetHighestBidForAuction(ctx, auctionID)
	if err != nil {
		fmt.Printf("[Mail] ℹ️ No bids found for auction %s. No emails to send.\n", auctionID)
		return nil
	}

	// Step 3: Fetch winning user info
	winner, err := s.authStore.GetUserByID(ctx, highestBid.SenderID)
	if err != nil || winner == nil {
		return fmt.Errorf("failed to get winner user: %w", err)
	}

	fmt.Printf("[Mail] 🏆 Found winner %s for auction %s with bid %.2f\n", winner.UserName, auctionID, highestBid.Amount)

	// Step 4: Get all bidders for this auction
	bids, err := s.bidStore.GetBidsByAuctionID(ctx, auctionID)
	if err != nil {
		return fmt.Errorf("failed to get auction bids: %w", err)
	}

	// Create a map to track unique bidders and successful email sends
	bidderMap := make(map[string]bool)
	emailsSent := 0
	emailsFailed := 0

	// Step 5: Send emails to all participants
	for _, bid := range bids {
		// Skip if we've already processed this bidder
		if bidderMap[bid.SenderID] {
			continue
		}
		bidderMap[bid.SenderID] = true

		// Get bidder info
		bidder, err := s.authStore.GetUserByID(ctx, bid.SenderID)
		if err != nil || bidder == nil {
			fmt.Printf("[Mail] ❌ Failed to get bidder info for ID %s: %v\n", bid.SenderID, err)
			emailsFailed++
			continue
		}

		// Send appropriate email based on whether they won or not
		if bidder.ID == winner.ID {
			// Send winner email
			fmt.Printf("[Mail] 🎉 Sending winner email to %s\n", bidder.Email)
			if err := s.sendWinnerEmail(ctx, bidder, auction, highestBid); err != nil {
				fmt.Printf("[Mail] ❌ Failed to send winner email to %s: %v\n", bidder.Email, err)
				emailsFailed++
			} else {
				fmt.Printf("[Mail] ✅ Winner email sent successfully to %s\n", bidder.Email)
				emailsSent++
			}
		} else {
			// Send participant email
			fmt.Printf("[Mail] 📨 Sending participant email to %s\n", bidder.Email)
			if err := s.sendParticipantEmail(ctx, bidder, auction, winner, highestBid); err != nil {
				fmt.Printf("[Mail] ❌ Failed to send participant email to %s: %v\n", bidder.Email, err)
				emailsFailed++
			} else {
				fmt.Printf("[Mail] ✅ Participant email sent successfully to %s\n", bidder.Email)
				emailsSent++
			}
		}
	}

	// Also notify the auction creator
	seller, err := s.authStore.GetUserByID(ctx, auction.User.ID)
	if err == nil && seller != nil && !bidderMap[seller.ID] {
		fmt.Printf("[Mail] 📤 Sending seller notification email to %s\n", seller.Email)
		if err := s.sendSellerEmail(ctx, seller, auction, winner, highestBid); err != nil {
			fmt.Printf("[Mail] ❌ Failed to send seller email to %s: %v\n", seller.Email, err)
			emailsFailed++
		} else {
			fmt.Printf("[Mail] ✅ Seller email sent successfully to %s\n", seller.Email)
			emailsSent++
		}
	}

	fmt.Printf("[Mail] 📊 Email summary for auction %s: %d sent, %d failed\n", auctionID, emailsSent, emailsFailed)

	// Return success even if some emails failed - we don't want to retry the entire batch
	return nil
}

// sendWinnerEmail sends a congratulatory email to the auction winner
func (s *MailService) sendWinnerEmail(ctx context.Context, winner *types.User, auction *types.AuctionData, bid *types.Bid) error {
	subject := fmt.Sprintf("🎉 You Won the Auction: %s!", auction.Title)
	body := fmt.Sprintf(`
	<!DOCTYPE html>
	<html>
	<head>
		<meta charset="UTF-8">
		<title>You Won!</title>
	</head>
	<body style="margin:0;padding:0;background-color:#fff8f0;font-family:Helvetica,Arial,sans-serif;">
		<div style="max-width:600px;margin:30px auto;padding:30px;background:white;border-radius:10px;box-shadow:0 0 10px rgba(0,0,0,0.1);">
			<h1 style="color:#ff6f00;text-align:center;">🎉 Congratulations, %s!</h1>
			<p style="font-size:16px;color:#333;">You have successfully won the auction:</p>

			<div style="background:#fff3e0;border-left:5px solid #ffa726;padding:15px;margin:20px 0;">
				<h2 style="margin:0;color:#e65100;">%s</h2>
				<p style="margin:5px 0;color:#555;">%s</p>
				<p style="margin:5px 0;"><strong>Winning Bid:</strong> ₹%.2f</p>
				<p style="margin:5px 0;"><strong>Auction Ended:</strong> %s</p>
			</div>

			<p style="font-size:15px;color:#444;">We will soon contact you with the next steps to finalize your purchase.</p>

			<hr style="border:none;border-top:1px solid #eee;margin:30px 0;">

			<p style="font-size:14px;color:#999;text-align:center;">
				Thank you for bidding with <strong>BidZy</strong><br>
				Stay tuned for more exciting auctions!
			</p>
		</div>
	</body>
	</html>
	`,
		winner.UserName,     // %s - name
		auction.Title,       // %s - title
		auction.Description, // %s - description
		bid.Amount,          // %.2f - winning amount
		auction.EndTime.Format("02 Jan 2006, 03:04 PM"), // %s - end date formatted
	)

	return s.SendEmail(ctx, EmailMessage{
		To:      winner.Email,
		Subject: subject,
		Body:    body,
		IsHTML:  true,
	})
}

// sendParticipantEmail sends a notification email to participants who didn't win
func (s *MailService) sendParticipantEmail(ctx context.Context, participant *types.User, auction *types.AuctionData, winner *types.User, winningBid *types.Bid) error {
	subject := fmt.Sprintf("Auction Ended: %s - Results", auction.Title)
	body := fmt.Sprintf(`
	<!DOCTYPE html>
	<html>
	<head>
		<meta charset="UTF-8">
		<title>Auction Results</title>
	</head>
	<body style="margin:0;padding:0;background-color:#f9f9f9;font-family:Helvetica,Arial,sans-serif;">
		<div style="max-width:600px;margin:30px auto;padding:30px;background:white;border-radius:10px;box-shadow:0 0 10px rgba(0,0,0,0.1);">
			<h1 style="color:#333;text-align:center;">Auction Results</h1>
			<p style="font-size:16px;color:#333;">Thank you for participating in the auction:</p>

			<div style="background:#f5f5f5;border-left:5px solid #9e9e9e;padding:15px;margin:20px 0;">
				<h2 style="margin:0;color:#333;">%s</h2>
				<p style="margin:5px 0;color:#555;">%s</p>
				<p style="margin:5px 0;"><strong>Auction Ended:</strong> %s</p>
			</div>

			<p style="font-size:15px;color:#444;">The auction has ended and the winning bid was ₹%.2f by %s.</p>

			<p style="font-size:15px;color:#444;">Better luck next time! We have many more exciting auctions coming up.</p>

			<hr style="border:none;border-top:1px solid #eee;margin:30px 0;">

			<p style="font-size:14px;color:#999;text-align:center;">
				Thank you for bidding with <strong>BidZy</strong><br>
				Stay tuned for more exciting auctions!
			</p>
		</div>
	</body>
	</html>
	`,
		auction.Title,       // %s - title
		auction.Description, // %s - description
		auction.EndTime.Format("02 Jan 2006, 03:04 PM"), // %s - end date formatted
		winningBid.Amount, // %.2f - winning amount
		winner.UserName,   // %s - winner name
	)

	return s.SendEmail(ctx, EmailMessage{
		To:      participant.Email,
		Subject: subject,
		Body:    body,
		IsHTML:  true,
	})
}

// sendSellerEmail sends a notification email to the seller about the auction results
func (s *MailService) sendSellerEmail(ctx context.Context, seller *types.User, auction *types.AuctionData, winner *types.User, winningBid *types.Bid) error {
	subject := fmt.Sprintf("Your Auction Has Ended: %s", auction.Title)
	body := fmt.Sprintf(`
	<!DOCTYPE html>
	<html>
	<head>
		<meta charset="UTF-8">
		<title>Auction Completed</title>
	</head>
	<body style="margin:0;padding:0;background-color:#f0f7ff;font-family:Helvetica,Arial,sans-serif;">
		<div style="max-width:600px;margin:30px auto;padding:30px;background:white;border-radius:10px;box-shadow:0 0 10px rgba(0,0,0,0.1);">
			<h1 style="color:#0277bd;text-align:center;">Auction Completed</h1>
			<p style="font-size:16px;color:#333;">Your auction has successfully ended:</p>

			<div style="background:#e1f5fe;border-left:5px solid #29b6f6;padding:15px;margin:20px 0;">
				<h2 style="margin:0;color:#01579b;">%s</h2>
				<p style="margin:5px 0;color:#555;">%s</p>
				<p style="margin:5px 0;"><strong>Final Selling Price:</strong> ₹%.2f</p>
				<p style="margin:5px 0;"><strong>Winner:</strong> %s</p>
				<p style="margin:5px 0;"><strong>Winner's Email:</strong> %s</p>
				<p style="margin:5px 0;"><strong>Auction Ended:</strong> %s</p>
			</div>

			<p style="font-size:15px;color:#444;">Please contact the buyer to arrange payment and delivery details.</p>

			<hr style="border:none;border-top:1px solid #eee;margin:30px 0;">

			<p style="font-size:14px;color:#999;text-align:center;">
				Thank you for using <strong>BidZy</strong><br>
				We hope to see more of your listings soon!
			</p>
		</div>
	</body>
	</html>
	`,
		auction.Title,       // %s - title
		auction.Description, // %s - description
		winningBid.Amount,   // %.2f - winning amount
		winner.UserName,     // %s - winner name
		winner.Email,        // %s - winner email
		auction.EndTime.Format("02 Jan 2006, 03:04 PM"), // %s - end date formatted
	)

	return s.SendEmail(ctx, EmailMessage{
		To:      seller.Email,
		Subject: subject,
		Body:    body,
		IsHTML:  true,
	})
}
