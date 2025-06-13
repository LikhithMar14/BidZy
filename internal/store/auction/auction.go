package auction

import (
	"context"
	"database/sql"
	"log"
	"github.com/LikhithMar14/BidZy/pkg/types"
)

type auctionStore struct {
	db *sql.DB
}

func NewAuctionRepository(db *sql.DB) *auctionStore {
	return &auctionStore{db: db}
}

func (s *auctionStore) CreateAuction(ctx context.Context, auction *types.CreateAuctionRequest, categoryIDs []int,userID string) (*types.CreateAuctionResponse, error) {
	log.Println("AUCTION FROM STORAGE LAYER:", auction)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	insertAuctionQuery := `
		INSERT INTO auctions (
			id, title, description, starting_price, current_price,
			start_date, end_date, status, image, user_id
		)
		VALUES ($1, $2, $3, $4, $4, $5, $6, $7, $8, $9)
		RETURNING id, title, starting_price, current_price, start_date, end_date, status, created_at;
	`

	var newAuction types.CreateAuctionResponse
	err = tx.QueryRowContext(ctx, insertAuctionQuery,
		auction.ID,
		auction.Title,
		auction.Description,
		auction.StartingPrice,
		auction.StartDateTime,
		auction.EndDateTime,
		auction.Status,
		auction.Image,
		userID,
	).Scan(
		&newAuction.ID,
		&newAuction.Title,
		&newAuction.StartingPrice,
		&newAuction.CurrentPrice,
		&newAuction.StartDateTime,
		&newAuction.EndDateTime,
		&newAuction.Status,
		&newAuction.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	// Insert into junction table
	insertCategoryQuery := `
		INSERT INTO auction_categories (auction_id, category_id)
		VALUES ($1, $2);
	`
	for _, categoryID := range categoryIDs {
		_, err := tx.ExecContext(ctx, insertCategoryQuery, auction.ID, categoryID)
		if err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &newAuction, nil
}
func (s *auctionStore) MarkAuctionsActive(ctx context.Context) error {
	query := `
		UPDATE auctions
		SET status = 'ACTIVE'
		WHERE start_date <= NOW() AND end_date >= NOW();
	`
	_, err := s.db.ExecContext(ctx, query)
	if err != nil {
		return err
	}
	return nil
}

func (s *auctionStore) MarkAuctionsEnded(ctx context.Context) error {
	query := `
		UPDATE auctions
		SET status = 'ENDED'
		WHERE end_date < NOW();
	`
	_, err := s.db.ExecContext(ctx, query)
	if err != nil {
		return err
	}
	return nil
}

// func (s *auctionStore) GetAuctionByID(ctx context.Context, id string) (*types.CreateAuctionResponse, error) {
// 	query := `
// 		SELECT a.id, a.title, a.description, a.starting_price, a.current_price, 
// 		       a.start_date, a.end_date, a.status, a.image, a.created_at, a.updated_at,
// 		       u.user_name as seller_name, u.email as seller_email
// 		FROM auctions a
// 		JOIN users u ON a.user_id = u.id
// 		WHERE a.id = $1;`

// 	var auction types.CreateAuctionResponse	
// 	err := s.db.QueryRowContext(ctx, query, id).Scan(
// 		&auction.ID,
// 		&auction.Title,
// 		&auction.Description,
// 		&auction.StartingPrice,
// 		&auction.CurrentPrice,
// 		&auction.StartDateTime,
// 		&auction.EndDateTime,
// 		&auction.Status,
// 		&auction.Image,
// 		&auction.CreatedAt,
// 		&auction.UpdatedAt,
// 		&auction.User.UserName,
// 		&auction.User.Email,
// 	)

// 	if err != nil {
// 		if err == sql.ErrNoRows {
// 			return nil, errors.New("auction not found")
// 		}
// 		return nil, err
// 	}

// 	return &auction, nil
// }

// func (s *auctionStore) GetAuctionByIDWithCategories(ctx context.Context, id string) (*types.AuctionWithCategories, error) {
// 	query := `
// 		SELECT a.id, a.title, a.description, a.starting_price, a.current_price, 
// 		       a.start_date, a.end_date, a.status, a.image, a.created_at, a.updated_at,
// 		       u.user_name as seller_name,
// 		       COALESCE(array_agg(c.name) FILTER (WHERE c.name IS NOT NULL), '{}') as categories
// 		FROM auctions a
// 		JOIN users u ON a.user_id = u.id
// 		LEFT JOIN auction_categories ac ON a.id = ac.auction_id
// 		LEFT JOIN categories c ON ac.category_id = c.id
// 		WHERE a.id = $1
// 		GROUP BY a.id, u.user_name;`

// 	var auction types.AuctionWithCategories
// 	err := s.db.QueryRowContext(ctx, query, id).Scan(
// 		&auction.ID,
// 		&auction.Title,
// 		&auction.Description,
// 		&auction.StartingPrice,
// 		&auction.CurrentPrice,
// 		&auction.StartDate,
// 		&auction.EndDate,
// 		&auction.Status,
// 		&auction.Image,
// 		&auction.CreatedAt,
// 		&auction.UpdatedAt,
// 		&auction.SellerName,
// 		pq.Array(&auction.Categories),
// 	)

// 	if err != nil {
// 		if err == sql.ErrNoRows {
// 			return nil, errors.New("auction not found")
// 		}
// 		return nil, err
// 	}

// 	return &auction, nil
// }

// func (s *auctionStore) GetAllActiveAuctions(ctx context.Context) ([]*types.AuctionSummary, error) {
// 	query := `
// 		SELECT a.id, a.title, a.description, a.starting_price, a.current_price, 
// 		       a.start_date, a.end_date, a.status, a.image, a.created_at,
// 		       u.user_name as seller_name,
// 		       (SELECT COUNT(*) FROM bids WHERE auction_id = a.id) as bid_count
// 		FROM auctions a
// 		JOIN users u ON a.user_id = u.id
// 		WHERE a.status = 'ACTIVE'
// 		ORDER BY a.created_at DESC;`

// 	rows, err := s.db.QueryContext(ctx, query)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer rows.Close()

// 	var auctions []*types.AuctionSummary
// 	for rows.Next() {
// 		auction := &types.AuctionSummary{}
// 		err := rows.Scan(
// 			&auction.ID,
// 			&auction.Title,
// 			&auction.Description,
// 			&auction.StartingPrice,
// 			&auction.CurrentPrice,
// 			&auction.StartDate,
// 			&auction.EndDate,
// 			&auction.Status,
// 			&auction.Image,
// 			&auction.CreatedAt,
// 			&auction.SellerName,
// 			&auction.BidCount,
// 		)
// 		if err != nil {
// 			return nil, err
// 		}
// 		auctions = append(auctions, auction)
// 	}

// 	if err = rows.Err(); err != nil {
// 		return nil, err
// 	}

// 	return auctions, nil
// }

// // func (s *auctionStore) GetAuctionsWithPagination(ctx context.Context, status string, limit, offset int) ([]*types.AuctionSummary, error) {
// 	query := `
// 		SELECT a.id, a.title, a.description, a.starting_price, a.current_price, 
// 		       a.start_date, a.end_date, a.status, a.image, a.created_at,
// 		       u.user_name as seller_name,
// 		       (SELECT COUNT(*) FROM bids WHERE auction_id = a.id) as bid_count
// 		FROM auctions a
// 		JOIN users u ON a.user_id = u.id
// 		WHERE a.status = $1
// 		ORDER BY a.created_at DESC
// 		LIMIT $2 OFFSET $3;`

// 	rows, err := s.db.QueryContext(ctx, query, status, limit, offset)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer rows.Close()

// 	var auctions []*types.AuctionSummary
// 	for rows.Next() {
// 		auction := &types.AuctionSummary{}
// 		err := rows.Scan(
// 			&auction.ID,
// 			&auction.Title,
// 			&auction.Description,
// 			&auction.StartingPrice,
// 			&auction.CurrentPrice,
// 			&auction.StartDate,
// 			&auction.EndDate,
// 			&auction.Status,
// 			&auction.Image,
// 			&auction.CreatedAt,
// 			&auction.SellerName,
// 			&auction.BidCount,
// 		)
// 		if err != nil {
// 			return nil, err
// 		}
// 		auctions = append(auctions, auction)
// 	}

// 	if err = rows.Err(); err != nil {
// 		return nil, err
// 	}

// 	return auctions, nil
// }

// func (s *auctionStore) GetAuctionsByUserID(ctx context.Context, userID string) ([]*types.CreateAuctionResponse, error) {
// 	query := `
// 		SELECT a.id, a.title, a.description, a.starting_price, a.current_price, 
// 		       a.start_date, a.end_date, a.status, a.image, a.created_at, a.updated_at,
// 		       (SELECT COUNT(*) FROM bids WHERE auction_id = a.id) as bid_count
// 		FROM auctions a
// 		WHERE a.user_id = $1
// 		ORDER BY a.created_at DESC;`

// 	rows, err := s.db.QueryContext(ctx, query, userID)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer rows.Close()

// 	var auctions []*types.CreateAuctionResponse
// 	for rows.Next() {
// 		auction := &types.CreateAuctionResponse{}
// 		err := rows.Scan(
// 			&auction.ID,
// 			&auction.Title,
// 			&auction.Description,
// 			&auction.StartingPrice,
// 			&auction.CurrentPrice,
// 			&auction.StartDateTime,
// 			&auction.EndDateTime,
// 			&auction.Status,
// 			&auction.Image,
// 			&auction.CreatedAt,
// 			&auction.UpdatedAt,
// 			&auction.BidCount,
// 		)
// 		if err != nil {
// 			return nil, err
// 		}
// 		auctions = append(auctions, auction)
// 	}

// 	if err = rows.Err(); err != nil {
// 		return nil, err
// 	}

// 	return auctions, nil
// }

// func (s *auctionStore) GetAuctionsByCategory(ctx context.Context, categoryID string) ([]*types.CreateAuctionResponse, error) {
// 	query := `
// 		SELECT a.id, a.title, a.description, a.starting_price, a.current_price, 
// 		       a.start_date, a.end_date, a.status, a.image, a.created_at,
// 		       u.user_name as seller_name
// 		FROM auctions a
// 		JOIN users u ON a.user_id = u.id
// 		JOIN auction_categories ac ON a.id = ac.auction_id
// 		WHERE ac.category_id = $1 AND a.status = 'ACTIVE'
// 		ORDER BY a.created_at DESC;`

// 	rows, err := s.db.QueryContext(ctx, query, categoryID)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer rows.Close()

// 	var auctions []*types.AuctionSummary
// 	for rows.Next() {
// 		auction := &types.AuctionSummary{}
// 		err := rows.Scan(
// 			&auction.ID,
// 			&auction.Title,
// 			&auction.Description,
// 			&auction.StartingPrice,
// 			&auction.CurrentPrice,
// 			&auction.StartDate,
// 			&auction.EndDate,
// 			&auction.Status,
// 			&auction.Image,
// 			&auction.CreatedAt,
// 			&auction.SellerName,
// 		)
// 		if err != nil {
// 			return nil, err
// 		}
// 		auctions = append(auctions, auction)
// 	}

// 	if err = rows.Err(); err != nil {
// 		return nil, err
// 	}

// 	return auctions, nil
// }

// func (s *auctionStore) SearchAuctions(ctx context.Context, searchTerm string) ([]*types.AuctionSummary, error) {
// 	query := `
// 		SELECT a.id, a.title, a.description, a.starting_price, a.current_price, 
// 		       a.start_date, a.end_date, a.status, a.image, a.created_at,
// 		       u.user_name as seller_name
// 		FROM auctions a
// 		JOIN users u ON a.user_id = u.id
// 		WHERE (a.title ILIKE '%' || $1 || '%' OR a.description ILIKE '%' || $1 || '%')
// 		  AND a.status = 'ACTIVE'
// 		ORDER BY a.created_at DESC;`

// 	rows, err := s.db.QueryContext(ctx, query, searchTerm)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer rows.Close()

// 	var auctions []*types.AuctionSummary
// 	for rows.Next() {
// 		auction := &types.AuctionSummary{}
// 		err := rows.Scan(
// 			&auction.ID,
// 			&auction.Title,
// 			&auction.Description,
// 			&auction.StartingPrice,
// 			&auction.CurrentPrice,
// 			&auction.StartDate,
// 			&auction.EndDate,
// 			&auction.Status,
// 			&auction.Image,
// 			&auction.CreatedAt,
// 			&auction.SellerName,
// 		)
// 		if err != nil {
// 			return nil, err
// 		}
// 		auctions = append(auctions, auction)
// 	}

// 	if err = rows.Err(); err != nil {
// 		return nil, err
// 	}

// 	return auctions, nil
// }

// func (s *auctionStore) GetAuctionsEndingSoon(ctx context.Context) ([]*types.EndingSoonAuction, error) {
// 	query := `
// 		SELECT a.id, a.title, a.current_price, a.end_date, a.image,
// 		       u.user_name as seller_name
// 		FROM auctions a
// 		JOIN users u ON a.user_id = u.id
// 		WHERE a.status = 'ACTIVE' 
// 		  AND a.end_date <= NOW() + INTERVAL '24 hours'
// 		  AND a.end_date > NOW()
// 		ORDER BY a.end_date ASC;`

// 	rows, err := s.db.QueryContext(ctx, query)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer rows.Close()

// 	var auctions []*types.EndingSoonAuction
// 	for rows.Next() {
// 		auction := &types.EndingSoonAuction{}
// 		err := rows.Scan(
// 			&auction.ID,
// 			&auction.Title,
// 			&auction.CurrentPrice,
// 			&auction.EndDate,
// 			&auction.Image,
// 			&auction.SellerName,
// 		)
// 		if err != nil {
// 			return nil, err
// 		}
// 		auctions = append(auctions, auction)
// 	}

// 	if err = rows.Err(); err != nil {
// 		return nil, err
// 	}

// 	return auctions, nil
// }

// func (s *auctionStore) GetRecentAuctions(ctx context.Context) ([]*types.RecentAuction, error) {
// 	query := `
// 		SELECT a.id, a.title, a.starting_price, a.current_price, a.status, a.image, a.created_at,
// 		       u.user_name as seller_name
// 		FROM auctions a
// 		JOIN users u ON a.user_id = u.id
// 		WHERE a.created_at >= NOW() - INTERVAL '7 days'
// 		ORDER BY a.created_at DESC;`

// 	rows, err := s.db.QueryContext(ctx, query)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer rows.Close()

// 	var auctions []*types.RecentAuction
// 	for rows.Next() {
// 		auction := &types.RecentAuction{}
// 		err := rows.Scan(
// 			&auction.ID,
// 			&auction.Title,
// 			&auction.StartingPrice,
// 			&auction.CurrentPrice,
// 			&auction.Status,
// 			&auction.Image,
// 			&auction.CreatedAt,
// 			&auction.SellerName,
// 		)
// 		if err != nil {
// 			return nil, err
// 		}
// 		auctions = append(auctions, auction)
// 	}

// 	if err = rows.Err(); err != nil {
// 		return nil, err
// 	}

// 	return auctions, nil
// }

// func (s *auctionStore) UpdateAuction(ctx context.Context, auctionID, userID string, updates *types.UpdateAuctionRequest) (*types.UpdateAuctionResponse, error) {
// 	query := `
// 		UPDATE auctions 
// 		SET title = $2, description = $3, starting_price = $4, start_date = $5, 
// 		    end_date = $6, status = $7, image = $8, updated_at = NOW()
// 		WHERE id = $1 AND user_id = $9
// 		RETURNING id, title, updated_at;`

// 	var response types.UpdateAuctionResponse
// 	err := s.db.QueryRowContext(ctx, query,
// 		auctionID,
// 		updates.Title,
// 		updates.Description,
// 		updates.StartingPrice,
// 		updates.StartDate,
// 		updates.EndDate,
// 		updates.Status,
// 		updates.Image,
// 		userID,
// 	).Scan(
// 		&response.ID,
// 		&response.Title,
// 		&response.UpdatedAt,
// 	)

// 	if err != nil {
// 		if err == sql.ErrNoRows {
// 			return nil, errors.New("auction not found or not authorized")
// 		}
// 		return nil, err
// 	}

// 	return &response, nil
// }

// func (s *auctionStore) UpdateAuctionStatus(ctx context.Context, auctionID, status string) (*types.UpdateStatusResponse, error) {
// 	query := `
// 		UPDATE auctions 
// 		SET status = $2, updated_at = NOW() 
// 		WHERE id = $1 
// 		RETURNING id, status, updated_at;`

// 	var response types.UpdateStatusResponse
// 	err := s.db.QueryRowContext(ctx, query, auctionID, status).Scan(
// 		&response.ID,
// 		&response.Status,
// 		&response.UpdatedAt,
// 	)

// 	if err != nil {
// 		if err == sql.ErrNoRows {
// 			return nil, errors.New("auction not found")
// 		}
// 		return nil, err
// 	}

// 	return &response, nil
// }

// func (s *auctionStore) UpdateAuctionCurrentPrice(ctx context.Context, auctionID string, currentPrice float64) (*types.UpdatePriceResponse, error) {
// 	query := `
// 		UPDATE auctions 
// 		SET current_price = $2, updated_at = NOW() 
// 		WHERE id = $1 
// 		RETURNING id, current_price, updated_at;`

// 	var response types.UpdatePriceResponse
// 	err := s.db.QueryRowContext(ctx, query, auctionID, currentPrice).Scan(
// 		&response.ID,
// 		&response.CurrentPrice,
// 		&response.UpdatedAt,
// 	)

// 	if err != nil {
// 		if err == sql.ErrNoRows {
// 			return nil, errors.New("auction not found")
// 		}
// 		return nil, err
// 	}

// 	return &response, nil
// }

// func (s *auctionStore) DeleteAuction(ctx context.Context, auctionID, userID string) error {
// 	query := `DELETE FROM auctions WHERE id = $1 AND user_id = $2;`

// 	result, err := s.db.ExecContext(ctx, query, auctionID, userID)
// 	if err != nil {
// 		return err
// 	}

// 	rowsAffected, err := result.RowsAffected()
// 	if err != nil {
// 		return err
// 	}

// 	if rowsAffected == 0 {
// 		return errors.New("auction not found or not authorized")
// 	}

// 	return nil
// }

// func (s *auctionStore) GetExpiredAuctions(ctx context.Context) ([]*types.ExpiredAuction, error) {
// 	query := `
// 		SELECT id, title, end_date 
// 		FROM auctions 
// 		WHERE status = 'ACTIVE' AND end_date <= NOW();`

// 	rows, err := s.db.QueryContext(ctx, query)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer rows.Close()

// 	var auctions []*types.ExpiredAuction
// 	for rows.Next() {
// 		auction := &types.ExpiredAuction{}
// 		err := rows.Scan(
// 			&auction.ID,
// 			&auction.Title,
// 			&auction.EndDate,
// 		)
// 		if err != nil {
// 			return nil, err
// 		}
// 		auctions = append(auctions, auction)
// 	}

// 	if err = rows.Err(); err != nil {
// 		return nil, err
// 	}

// 	return auctions, nil
// }



func (s *auctionStore) AddCategoryToAuction(ctx context.Context, auctionID, categoryID string) error {
	query := `
		INSERT INTO auction_categories (auction_id, category_id) 
		VALUES ($1, $2)
		ON CONFLICT (auction_id, category_id) DO NOTHING;`

	_, err := s.db.ExecContext(ctx, query, auctionID, categoryID)
	return err
}

func (s *auctionStore) RemoveCategoryFromAuction(ctx context.Context, auctionID, categoryID string) error {
	query := `DELETE FROM auction_categories WHERE auction_id = $1 AND category_id = $2;`

	_, err := s.db.ExecContext(ctx, query, auctionID, categoryID)
	return err
}

// func (s *auctionStore) GetCategoriesForAuction(ctx context.Context, auctionID string) ([]*types.Category, error) {
// 	query := `
// 		SELECT c.id, c.name 
// 		FROM categories c
// 		JOIN auction_categories ac ON c.id = ac.category_id
// 		WHERE ac.auction_id = $1;`

// 	rows, err := s.db.QueryContext(ctx, query, auctionID)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer rows.Close()

// 	var categories []*types.Category
// 	for rows.Next() {
// 		category := &types.Category{}
// 		err := rows.Scan(&category.ID, &category.Name)
// 		if err != nil {
// 			return nil, err
// 		}
// 		categories = append(categories, category)
// 	}

// 	if err = rows.Err(); err != nil {
// 		return nil, err
// 	}

// 	return categories, nil
// }

func (s *auctionStore) RemoveAllCategoriesFromAuction(ctx context.Context, auctionID string) error {
	query := `DELETE FROM auction_categories WHERE auction_id = $1;`

	_, err := s.db.ExecContext(ctx, query, auctionID)
	return err
}