package category

import (
	"context"

	"github.com/LikhithMar14/BidZy/internal/store"
	"github.com/LikhithMar14/BidZy/pkg/types"
)

type CategoryService struct {
	store store.CategoryRepository
}

func NewCategoryService(store store.CategoryRepository) *CategoryService {
	return &CategoryService{store: store}
}

func (s *CategoryService) GetAllCategories(ctx context.Context) ([]*types.Category, error) {
	return s.store.GetAllCategories(ctx)
}
