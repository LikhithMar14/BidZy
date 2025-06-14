package auction

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/LikhithMar14/BidZy/pkg/types"
	"log"
)

type auctionStore struct {
	db *sql.DB
}

func NewAuctionRepository(db *sql.DB) *auctionStore {
	return &auctionStore{db: db}
}

func (s *auctionStore) CreateAuction(ctx context.Context, auction *types.CreateAuctionRequest, categoryIDs []int, userID string) (*types.AuctionData, error) {
	log.Println("AUCTION FROM STORAGE LAYER:", auction)
	log.Println("USER ID in CreateAuction:", userID)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	log.Println("Status received in handler/service:", auction.Status)

	defer tx.Rollback()

	insertAuctionQuery := `
		INSERT INTO auctions (
			title, description, starting_price, current_price,
			increment, start_date, end_date, status, image, user_id
		)
		VALUES ($1, $2, $3, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, title, description, starting_price, current_price,
		          start_date, end_date, increment, status, image, user_id;
	`

	var newAuction types.AuctionData

	err = tx.QueryRowContext(ctx, insertAuctionQuery,
		auction.Title,
		auction.Description,
		auction.StartingPrice,
		auction.Increment,
		auction.StartDateTime,
		auction.EndDateTime,
		auction.Status,
		auction.Image,
		userID,
	).Scan(
		&newAuction.AuctionID,
		&newAuction.Title,
		&newAuction.Description,
		&newAuction.StartingPrice,
		&newAuction.CurrentPrice,
		&newAuction.StartTime,
		&newAuction.EndTime,
		&newAuction.Increment,
		&newAuction.Status,
		&newAuction.Image,
		&newAuction.User.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert auction: %w", err)
	}

	newAuction.IsActive = newAuction.Status == "ACTIVE"
	newAuction.ClientCount = 0
	newAuction.HighestBidder = ""

	// Insert into junction table
	if len(categoryIDs) > 0 {
		insertCategoryQuery := `
			INSERT INTO auction_categories (auction_id, category_id)
			VALUES ($1, $2);
		`
		for _, categoryID := range categoryIDs {
			if _, err := tx.ExecContext(ctx, insertCategoryQuery, newAuction.AuctionID, categoryID); err != nil {
				return nil, fmt.Errorf("failed to insert category: %w", err)
			}
		}
	}
	log.Println("AUCTION FROM STORAGE LAYER:", newAuction)

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	// ✅ Corrected user fetch query
	userQuery := `
		SELECT id, user_name, email, created_at, updated_at 
		FROM users WHERE id = $1
	`
	err = s.db.QueryRowContext(ctx, userQuery, userID).Scan(
		&newAuction.User.ID,
		&newAuction.User.UserName,
		&newAuction.User.Email,
		&newAuction.User.CreatedAt,
		&newAuction.User.UpdatedAt,
	)
	if err != nil {
		log.Println("ERROR IN FETCHING USER:", err)
		return nil, fmt.Errorf("failed to fetch user: %w", err)
	}

	// Fetch category IDs
	catQuery := `SELECT category_id FROM auction_categories WHERE auction_id = $1`
	rows, err := s.db.QueryContext(ctx, catQuery, newAuction.AuctionID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch categories: %w", err)
	}
	defer rows.Close()

	newAuction.CategoryIDs = []int{}
	for rows.Next() {
		var catID int
		if err := rows.Scan(&catID); err != nil {
			return nil, fmt.Errorf("failed to scan category_id: %w", err)
		}
		newAuction.CategoryIDs = append(newAuction.CategoryIDs, catID)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating categories: %w", err)
	}

	fmt.Println("++++NEW AUCTION++++", newAuction)
	return &newAuction, nil
}


func (s *auctionStore) MarkAuctionsActive(ctx context.Context) error {
	query := `
		UPDATE auctions
		SET status = 'ACTIVE'
		WHERE start_date <= (NOW() AT TIME ZONE 'UTC')
		  AND end_date > (NOW() AT TIME ZONE 'UTC')
		  AND status = 'INACTIVE';
	`
	_, err := s.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to mark auctions active: %w", err)
	}

	return nil
}

func (s *auctionStore) MarkAuctionsEnded(ctx context.Context) error {
	query := `
		UPDATE auctions
		SET status = 'ENDED'
		WHERE end_date <= (NOW() AT TIME ZONE 'UTC')
			AND status IN ('ACTIVE');
	`
	_, err := s.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to mark auctions ended: %w", err)
	}

	return nil
}

func (s *auctionStore) GetAllAuctions(ctx context.Context) ([]*types.AuctionData, error) {
	query := `
		SELECT 
			a.id, a.title, a.description, a.starting_price, a.current_price,
			a.start_date, a.end_date, a.status, 
			COALESCE(a.increment, 100) as increment, 
			a.image,
			u.id, u.user_name, u.email, u.created_at, u.updated_at
		FROM auctions a
		LEFT JOIN users u ON a.user_id = u.id
		ORDER BY a.created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query auctions: %w", err)
	}
	defer rows.Close()

	var auctions []*types.AuctionData

	for rows.Next() {
		var auction types.AuctionData
		var status string
		var userID, userName, userEmail sql.NullString
		var userCreatedAt, userUpdatedAt sql.NullTime

		err := rows.Scan(
			&auction.AuctionID,
			&auction.Title,
			&auction.Description,
			&auction.StartingPrice,
			&auction.CurrentPrice,
			&auction.StartTime,
			&auction.EndTime,
			&status,
			&auction.Increment,
			&auction.Image,
			&userID,
			&userName,
			&userEmail,
			&userCreatedAt,
			&userUpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan auction row: %w", err)
		}

		// Set status and isActive
		auction.Status = status
		auction.IsActive = status == "ACTIVE"

		// Set user info if present
		if userID.Valid {
			auction.User = types.User{
				ID:        userID.String,
				UserName:  userName.String,
				Email:     userEmail.String,
				CreatedAt: userCreatedAt.Time,
				UpdatedAt: userUpdatedAt.Time,
			}
		}

		// Set defaults for client count and highest bidder
		auction.ClientCount = 0
		auction.HighestBidder = ""

		// Empty category slice (filled later)
		auction.CategoryIDs = []int{}

		auctions = append(auctions, &auction)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating auction rows: %w", err)
	}

	// Efficiently populate category IDs
	if len(auctions) > 0 {
		err = s.populateAuctionCategories(ctx, auctions)
		if err != nil {
			return nil, fmt.Errorf("failed to populate categories: %w", err)
		}
	}

	return auctions, nil
}

// Helper function to populate categories for multiple auctions efficiently
func (s *auctionStore) populateAuctionCategories(ctx context.Context, auctions []*types.AuctionData) error {
	if len(auctions) == 0 {
		return nil
	}

	// Create a map of auction ID to auction for quick lookup
	auctionMap := make(map[string]*types.AuctionData)
	auctionIDs := make([]interface{}, len(auctions))

	for i, auction := range auctions {
		auctionMap[auction.AuctionID] = auction
		auctionIDs[i] = auction.AuctionID
	}

	// Build placeholders for IN clause
	placeholders := ""
	for i := range auctionIDs {
		if i > 0 {
			placeholders += ", "
		}
		placeholders += fmt.Sprintf("$%d", i+1)
	}

	query := fmt.Sprintf(`
		SELECT auction_id, category_id 
		FROM auction_categories 
		WHERE auction_id IN (%s)
	`, placeholders)

	rows, err := s.db.QueryContext(ctx, query, auctionIDs...)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var auctionID string
		var categoryID int
		if err := rows.Scan(&auctionID, &categoryID); err != nil {
			return err
		}

		if auction, exists := auctionMap[auctionID]; exists {
			auction.CategoryIDs = append(auction.CategoryIDs, categoryID)
		}
	}

	return rows.Err()
}

func (s *auctionStore) AddCategoryToAuction(ctx context.Context, auctionID, categoryID string) error {
	query := `
		INSERT INTO auction_categories (auction_id, category_id) 
		VALUES ($1, $2)
		ON CONFLICT (auction_id, category_id) DO NOTHING;`

	_, err := s.db.ExecContext(ctx, query, auctionID, categoryID)
	if err != nil {
		return fmt.Errorf("failed to add category to auction: %w", err)
	}
	return nil
}

func (s *auctionStore) GetAuctionByID(ctx context.Context, auctionID string) (*types.AuctionData, error) {
	query := `
		SELECT 
			a.id, a.title, a.description, a.starting_price, a.current_price, 
			a.start_date, a.end_date, a.status, 
			COALESCE(a.increment, 100) as increment, 
			a.image,
			u.id, u.user_name, u.email, u.created_at, u.updated_at
		FROM auctions a
		LEFT JOIN users u ON a.user_id = u.id
		WHERE a.id = $1;
	`

	row := s.db.QueryRowContext(ctx, query, auctionID)
	var auction types.AuctionData
	var status string
	var userID, userName, userEmail sql.NullString
	var userCreatedAt, userUpdatedAt sql.NullTime

	err := row.Scan(
		&auction.AuctionID,
		&auction.Title,
		&auction.Description,
		&auction.StartingPrice,
		&auction.CurrentPrice,
		&auction.StartTime,
		&auction.EndTime,
		&auction.Status,
		&auction.Increment,
		&auction.Image,
		&userID,
		&userName,
		&userEmail,
		&userCreatedAt,
		&userUpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("auction not found")
		}
		return nil, fmt.Errorf("failed to scan auction: %w", err)
	}

	auction.IsActive = status == "ACTIVE"
	auction.ClientCount = 0 // Initialize

	// Handle nullable user fields
	if userID.Valid {
		auction.User = types.User{
			ID:        userID.String,
			UserName:  userName.String,
			Email:     userEmail.String,
			CreatedAt: userCreatedAt.Time,
			UpdatedAt: userUpdatedAt.Time,
		}
	} else {
		auction.User = types.User{}
	}

	// Fetch categories
	categoryQuery := `SELECT category_id FROM auction_categories WHERE auction_id = $1;`
	rows, err := s.db.QueryContext(ctx, categoryQuery, auctionID)
	if err != nil {
		return nil, fmt.Errorf("failed to query categories: %w", err)
	}
	defer rows.Close()

	auction.CategoryIDs = []int{} // Initialize empty slice
	for rows.Next() {
		var cid int
		if err := rows.Scan(&cid); err != nil {
			return nil, fmt.Errorf("failed to scan category: %w", err)
		}
		auction.CategoryIDs = append(auction.CategoryIDs, cid)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating categories: %w", err)
	}

	return &auction, nil
}

func (s *auctionStore) GetAuctionsByUserID(ctx context.Context, userID string) ([]*types.AuctionData, error) {
	query := `
		SELECT 
			a.id, a.title, a.description, a.starting_price, a.current_price, 
			a.start_date, a.end_date, a.status, 
			COALESCE(a.increment, 100) as increment, 
			a.image,
			u.id, u.user_name, u.email, u.created_at, u.updated_at
		FROM auctions a
		LEFT JOIN users u ON a.user_id = u.id
		WHERE a.user_id = $1
		ORDER BY a.created_at DESC;
	`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query user auctions: %w", err)
	}
	defer rows.Close()

	var auctions []*types.AuctionData

	for rows.Next() {
		var auction types.AuctionData
		var status string
		var userIDResult, userName, userEmail sql.NullString
		var userCreatedAt, userUpdatedAt sql.NullTime

		if err := rows.Scan(
			&auction.AuctionID,
			&auction.Title,
			&auction.Description,
			&auction.StartingPrice,
			&auction.CurrentPrice,
			&auction.StartTime,
			&auction.EndTime,
			&status,
			&auction.Increment,
			&auction.Image,
			&userIDResult,
			&userName,
			&userEmail,
			&userCreatedAt,
			&userUpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan auction: %w", err)
		}

		auction.IsActive = status == "ACTIVE"
		auction.ClientCount = 0

		// Handle user data
		if userIDResult.Valid {
			auction.User = types.User{
				ID:        userIDResult.String,
				UserName:  userName.String,
				Email:     userEmail.String,
				CreatedAt: userCreatedAt.Time,
				UpdatedAt: userUpdatedAt.Time,
			}
		} else {
			auction.User = types.User{}
		}

		auction.CategoryIDs = []int{} // Initialize
		auctions = append(auctions, &auction)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating user auctions: %w", err)
	}

	// Populate categories for all auctions
	if len(auctions) > 0 {
		err = s.populateAuctionCategories(ctx, auctions)
		if err != nil {
			return nil, fmt.Errorf("failed to populate categories: %w", err)
		}
	}

	return auctions, nil
}

func (s *auctionStore) RemoveCategoryFromAuction(ctx context.Context, auctionID, categoryID string) error {
	query := `DELETE FROM auction_categories WHERE auction_id = $1 AND category_id = $2;`

	result, err := s.db.ExecContext(ctx, query, auctionID, categoryID)
	if err != nil {
		return fmt.Errorf("failed to remove category from auction: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no category found to remove")
	}

	return nil
}

func (s *auctionStore) RemoveAllCategoriesFromAuction(ctx context.Context, auctionID string) error {
	query := `DELETE FROM auction_categories WHERE auction_id = $1;`

	_, err := s.db.ExecContext(ctx, query, auctionID)
	if err != nil {
		return fmt.Errorf("failed to remove all categories from auction: %w", err)
	}
	return nil
}
