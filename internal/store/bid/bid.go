package bid

import (
	"context"
	"database/sql"
	"errors"
	"github.com/LikhithMar14/BidZy/pkg/types"
)

type bidStore struct {
	db *sql.DB
}

func NewBidRepository(db *sql.DB) *bidStore {
	return &bidStore{
		db: db,
	}
}

func (s *bidStore) PlaceBid(ctx context.Context, bid *types.NewBidRequest) (*types.NewBidResponse, error) {
	query := `INSERT INTO bids (amount, user_id, auction_id) 
	VALUES ($1, $2, $3) 
	RETURNING id, amount, created_at;`

	var newBid types.NewBidResponse
	err := s.db.QueryRowContext(ctx, query, bid.Amount, bid.SenderID, bid.AuctionID).Scan(&newBid.ID, &newBid.Amount, &newBid.CreatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("no rows found")
		}
		return nil, err
	}

	return &newBid, nil
}

func (s *bidStore) GetBidByID(ctx context.Context, id string) (*types.Bid, error) {
	query := `
		SELECT b.id, b.amount, b.created_at, b.user_id, b.auction_id,
		       u.user_name as bidder_name
		FROM bids b
		JOIN users u ON b.user_id = u.id
		WHERE b.id = $1;`

	var bid types.Bid
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&bid.ID, 
		&bid.Amount, 
		&bid.CreatedAt, 
		&bid.SenderID, 
		&bid.AuctionID, 
		&bid.BidderName,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("bid not found")
		}
		return nil, err
	}

	return &bid, nil
}

func (s *bidStore) GetBidsByAuctionID(ctx context.Context, auctionID string) ([]*types.Bid, error) {
	query := `
		SELECT b.id, b.amount, b.created_at, u.user_name as bidder_name
		FROM bids b
		JOIN users u ON b.user_id = u.id
		WHERE b.auction_id = $1
		ORDER BY b.amount DESC, b.created_at DESC;`

	rows, err := s.db.QueryContext(ctx, query, auctionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bids []*types.Bid
	for rows.Next() {
		bid := &types.Bid{}
		err := rows.Scan(
			&bid.ID,
			&bid.Amount,
			&bid.CreatedAt,
			&bid.BidderName,
		)
		if err != nil {
			return nil, err
		}
		bid.AuctionID = auctionID
		bids = append(bids, bid)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return bids, nil
}

func (s *bidStore) GetHighestBidForAuction(ctx context.Context, auctionID string) (*types.Bid, error) {
	query := `
		SELECT b.id, b.amount, b.created_at, b.user_id, u.user_name as bidder_name
		FROM bids b
		JOIN users u ON b.user_id = u.id
		WHERE b.auction_id = $1
		ORDER BY b.amount DESC, b.created_at DESC
		LIMIT 1;`

	var bid types.Bid
	err := s.db.QueryRowContext(ctx, query, auctionID).Scan(
		&bid.ID,
		&bid.Amount,
		&bid.CreatedAt,
		&bid.SenderID,
		&bid.BidderName,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("no bids found for auction")
		}
		return nil, err
	}

	bid.AuctionID = auctionID
	return &bid, nil
}

func (s *bidStore) GetUserBids(ctx context.Context, userID string) ([]*types.Bid, error) {
	query := `
		SELECT b.id, b.amount, b.created_at, 
		       a.id as auction_id, a.title as auction_title, a.status as auction_status,
		       a.end_date, a.image as auction_image
		FROM bids b
		JOIN auctions a ON b.auction_id = a.id
		WHERE b.user_id = $1
		ORDER BY b.created_at DESC;`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var userBids []*types.Bid
	for rows.Next() {
		userBid := &types.Bid{}
		err := rows.Scan(
			&userBid.ID,
			&userBid.Amount,
			&userBid.CreatedAt,
			&userBid.AuctionID,
			&userBid.AuctionTitle,
			&userBid.AuctionStatus,
			&userBid.AuctionEndDate,
			&userBid.AuctionImage,
		)
		if err != nil {
			return nil, err
		}
		userBids = append(userBids, userBid)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return userBids, nil
}

func (s *bidStore) GetBidCount(ctx context.Context, auctionID string) (int, error) {
	query := `SELECT COUNT(*) as bid_count FROM bids WHERE auction_id = $1;`

	var count int
	err := s.db.QueryRowContext(ctx, query, auctionID).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (s *bidStore) GetBidHistoryWithPagination(ctx context.Context, auctionID string, limit, offset int) ([]*types.Bid, error) {
	query := `
		SELECT b.id, b.amount, b.created_at, u.user_name as bidder_name
		FROM bids b
		JOIN users u ON b.user_id = u.id
		WHERE b.auction_id = $1
		ORDER BY b.created_at DESC
		LIMIT $2 OFFSET $3;`

	rows, err := s.db.QueryContext(ctx, query, auctionID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bids []*types.Bid
	for rows.Next() {
		bid := &types.Bid{}
		err := rows.Scan(
			&bid.ID,
			&bid.Amount,
			&bid.CreatedAt,
			&bid.BidderName,
		)
		if err != nil {
			return nil, err
		}
		bid.AuctionID = auctionID
		bids = append(bids, bid)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return bids, nil
}

func (s *bidStore) HasUserBidOnAuction(ctx context.Context, userID, auctionID string) (bool, error) {
	query := `
		SELECT EXISTS(
		  SELECT 1 FROM bids 
		  WHERE user_id = $1 AND auction_id = $2
		);`

	var exists bool
	err := s.db.QueryRowContext(ctx, query, userID, auctionID).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}

func (s *bidStore) GetUserHighestBidOnAuction(ctx context.Context, userID, auctionID string) (float64, error) {
	query := `
		SELECT COALESCE(MAX(amount), 0) as highest_bid
		FROM bids
		WHERE user_id = $1 AND auction_id = $2;`

	var highestBid float64
	err := s.db.QueryRowContext(ctx, query, userID, auctionID).Scan(&highestBid)
	if err != nil {
		return 0, err
	}

	return highestBid, nil
}