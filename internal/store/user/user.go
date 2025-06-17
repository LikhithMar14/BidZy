package user

import (
	"context"
	"database/sql"
	"github.com/LikhithMar14/BidZy/pkg/types"
)

type userStore struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *userStore {
	return &userStore{db: db}
}




func (s *userStore) GetUserByID(ctx context.Context, userID string) (*types.User, error) {
	query := `
		SELECT id, user_name, email, created_at, updated_at
		FROM users
		WHERE id = $1
	`
	var u types.User
	err := s.db.QueryRowContext(ctx, query, userID).Scan(
		&u.ID, &u.UserName, &u.Email, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *userStore) GetAuctionsByUserID(ctx context.Context, userID string) ([]*types.Auction, error) {
	query := `
		SELECT id, title, description, starting_price, current_price, increment,
		       start_date, end_date, status, image, client_count, created_at, updated_at
		FROM auctions
		WHERE user_id = $1
		ORDER BY created_at DESC
	`
	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var auctions []*types.Auction
	for rows.Next() {
		var a types.Auction
		err := rows.Scan(
			&a.ID, &a.Title, &a.Description, &a.StartingPrice, &a.CurrentPrice, &a.Increment,
			&a.StartDate, &a.EndDate, &a.Status, &a.Image, &a.ClientCount, &a.CreatedAt, &a.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		auctions = append(auctions, &a)
	}
	return auctions, nil
}


func (s *userStore) GetBidsByUserID(ctx context.Context, userID string) ([]*types.Bid, error) {
	query := `
		SELECT 
			b.id, b.amount, b.created_at, b.user_id, b.auction_id, u.user_name
		FROM bids b
		JOIN users u ON b.user_id = u.id
		WHERE b.user_id = $1
		ORDER BY b.created_at DESC
	`
	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bids []*types.Bid
	for rows.Next() {
		var b types.Bid
		err := rows.Scan(
			&b.ID, &b.Amount, &b.CreatedAt, &b.SenderID, &b.AuctionID, &b.BidderName,
		)
		if err != nil {
			return nil, err
		}
		bids = append(bids, &b)
	}
	return bids, nil
}


func (s *userStore) GetUserStats(ctx context.Context, userID string) (*types.UserStats, error) {
	query := `
		SELECT 
			(SELECT COUNT(*) FROM auctions WHERE user_id = $1),                             -- Auctions created
			(SELECT COUNT(*) FROM bids WHERE user_id = $1),                                -- Total bids
			(SELECT COALESCE(SUM(amount), 0) FROM bids WHERE user_id = $1),                -- Total amount bid
			(SELECT COUNT(*) FROM auctions WHERE user_id = $1 AND status = 'ACTIVE'),      -- Active auctions
			(SELECT COUNT(DISTINCT auction_id) FROM bids WHERE user_id = $1),              -- Participated auctions
			(SELECT COUNT(*) FROM auctions a 
			 WHERE a.status = 'ENDED' AND 
			       EXISTS (
			           SELECT 1 FROM bids b
			           WHERE b.auction_id = a.id 
			           AND b.user_id = $1 
			           AND b.amount = (
			               SELECT MAX(amount) FROM bids WHERE auction_id = a.id
			           )
			       )
			),                                                                             -- Won auctions
			(SELECT COALESCE(AVG(amount), 0) FROM bids WHERE user_id = $1),                -- Avg bid amount
			(SELECT COALESCE(MAX(amount), 0) FROM bids WHERE user_id = $1)                 -- Highest bid placed
	`

	var stats types.UserStats
	err := s.db.QueryRowContext(ctx, query, userID).Scan(
		&stats.AuctionsCreated,
		&stats.TotalBids,
		&stats.TotalAmountBid,
		&stats.ActiveAuctions,
		&stats.ParticipatedAuctions,
		&stats.WonAuctions,
		&stats.AvgBidAmount,
		&stats.HighestBidPlaced,
	)
	if err != nil {
		return nil, err
	}
	return &stats, nil
}


func (s *userStore) GetParticipatedAuctions(ctx context.Context, userID string) ([]*types.Auction, error) {
	query := `
		SELECT DISTINCT ON (a.id)
			a.id, a.title, a.description, a.starting_price, a.current_price, a.increment,
			a.start_date, a.end_date, a.status, a.image, a.client_count,
			a.created_at, a.updated_at
		FROM bids b
		JOIN auctions a ON b.auction_id = a.id
		WHERE b.user_id = $1
		ORDER BY a.id, b.created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var auctions []*types.Auction
	for rows.Next() {
		var a types.Auction
		err := rows.Scan(
			&a.ID, &a.Title, &a.Description, &a.StartingPrice, &a.CurrentPrice, &a.Increment,
			&a.StartDate, &a.EndDate, &a.Status, &a.Image, &a.ClientCount,
			&a.CreatedAt, &a.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		auctions = append(auctions, &a)
	}
	return auctions, nil
}
