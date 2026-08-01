// Package review implements product reviews: creation guarded by order
// ownership + completed status + one-review-per-order-item, and listing
// (FR-6.5).
package review

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"erdinhrmwn/bangunin/internal/domain/entity"
	"erdinhrmwn/bangunin/internal/domain/repository"
	"erdinhrmwn/bangunin/pkg/apperr"
)

type Usecase struct {
	reviews  repository.ReviewRepository
	orders   repository.OrderRepository
	variants repository.ProductVariantRepository
	products repository.ProductRepository
}

func New(
	reviews repository.ReviewRepository,
	orders repository.OrderRepository,
	variants repository.ProductVariantRepository,
	products repository.ProductRepository,
) *Usecase {
	return &Usecase{reviews: reviews, orders: orders, variants: variants, products: products}
}

// Create validates ownership, order status, and duplicate before inserting,
// then recomputes the product's denormalized rating (FR-6.5).
func (u *Usecase) Create(ctx context.Context, userID, orderItemID uuid.UUID, rating int, comment string) (*entity.Review, error) {
	item, err := u.orders.FindItemByID(ctx, orderItemID)
	if err != nil {
		return nil, err
	}
	o, err := u.orders.FindByID(ctx, item.OrderID)
	if err != nil {
		return nil, err
	}
	if o.UserID != userID {
		return nil, apperr.New("FORBIDDEN", "Order does not belong to you", 403)
	}
	if o.Status != entity.OrderStatusCompleted {
		return nil, apperr.New("VALIDATION_ERROR", "Order must be completed before it can be reviewed", 422)
	}

	exists, err := u.reviews.ExistsByOrderItemID(ctx, orderItemID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, apperr.New("CONFLICT", "This order item has already been reviewed", 409)
	}

	variant, err := u.variants.FindByID(ctx, item.VariantID)
	if err != nil {
		return nil, err
	}

	id, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("review: generate id: %w", err)
	}
	rv := &entity.Review{
		ID: id, OrderItemID: orderItemID, ProductID: variant.ProductID, UserID: userID,
		Rating: rating, Comment: comment,
	}
	if err := u.reviews.Create(ctx, rv); err != nil {
		return nil, err
	}

	avg, count, err := u.reviews.AverageRating(ctx, variant.ProductID)
	if err != nil {
		return nil, fmt.Errorf("review: compute average rating: %w", err)
	}
	if err := u.products.UpdateRating(ctx, variant.ProductID, avg, count); err != nil {
		return nil, fmt.Errorf("review: update product rating: %w", err)
	}
	return rv, nil
}

func (u *Usecase) ListByProduct(ctx context.Context, productID uuid.UUID, rating int, cursor string, limit int) ([]*entity.Review, string, error) {
	return u.reviews.ListByProduct(ctx, productID, rating, cursor, limit)
}

func (u *Usecase) ListByUser(ctx context.Context, userID uuid.UUID, page, perPage int) ([]*entity.Review, int, error) {
	return u.reviews.ListByUser(ctx, userID, page, perPage)
}
