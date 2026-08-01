package repository

import (
	"context"

	"github.com/google/uuid"

	"erdinhrmwn/bangunin/internal/domain/entity"
)

type ReviewRepository interface {
	Create(ctx context.Context, r *entity.Review) error
	ExistsByOrderItemID(ctx context.Context, orderItemID uuid.UUID) (bool, error)
	ListByProduct(ctx context.Context, productID uuid.UUID, rating int, cursor string, limit int) ([]*entity.Review, string, error)
	ListByUser(ctx context.Context, userID uuid.UUID, page, perPage int) ([]*entity.Review, int, error)
	// AverageRating computes the current avg+count for a product — called
	// after Create to recompute Product.RatingAvg/RatingCount.
	AverageRating(ctx context.Context, productID uuid.UUID) (avg float64, count int, err error)
}
