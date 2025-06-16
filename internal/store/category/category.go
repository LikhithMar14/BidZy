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
	fmt.Println("I AM HERE 1")
	query := `SELECT id, name, created_at FROM categories ORDER BY name ASC`
	rows, err := s.db.QueryContext(ctx, query)
	fmt.Println("rows ", rows)
	fmt.Println("err ", err)
	if err != nil {
		return nil, fmt.Errorf("querying categories: %w", err)
	}

	defer rows.Close()

	var categories []*types.Category
	fmt.Println("I AM HERE 2")
	for rows.Next() {
		c := &types.Category{}
		if err := rows.Scan(&c.ID, &c.Name, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning category: %w", err)
		}
		categories = append(categories, c)
	}
	fmt.Println("I AM HERE 3")

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}
	fmt.Println("I AM HERE 4")
	return categories, nil
}
