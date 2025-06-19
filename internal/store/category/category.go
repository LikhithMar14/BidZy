package category

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/LikhithMar14/BidZy/pkg/types"
)

type categoryStore struct {
	db *sql.DB
}

func NewCategoryRepository(db *sql.DB) *categoryStore {
	return &categoryStore{db: db}
}

func (s *categoryStore) GetAllCategories(ctx context.Context) ([]*types.Category, error) {
	query := `SELECT id, name, created_at FROM categories ORDER BY name ASC`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("querying categories: %w", err)
	}

	defer rows.Close()

	var categories []*types.Category
	for rows.Next() {
		c := &types.Category{}
		if err := rows.Scan(&c.ID, &c.Name, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning category: %w", err)
		}
		categories = append(categories, c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}
	return categories, nil
}
