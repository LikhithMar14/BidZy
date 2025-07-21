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

func (s *categoryStore) CreateCategory(ctx context.Context, name string) (*types.Category, error) {
	query := `INSERT INTO categories (name) VALUES ($1) RETURNING id, name, created_at`

	var category types.Category
	err := s.db.QueryRowContext(ctx, query, name).Scan(&category.ID, &category.Name, &category.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating category: %w", err)
	}

	return &category, nil
}

func (s *categoryStore) GetCategoryByName(ctx context.Context, name string) (*types.Category, error) {
	query := `SELECT id, name, created_at FROM categories WHERE name = $1`

	var category types.Category
	err := s.db.QueryRowContext(ctx, query, name).Scan(&category.ID, &category.Name, &category.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Category not found
		}
		return nil, fmt.Errorf("getting category by name: %w", err)
	}

	return &category, nil
}
