package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"erdinhrmwn/bangunin/internal/domain/entity"
	"erdinhrmwn/bangunin/internal/domain/errs"
)

type ReviewRepository struct {
	db *pgxpool.Pool
}

func NewReviewRepository(db *pgxpool.Pool) *ReviewRepository {
	return &ReviewRepository{db: db}
}

func (r *ReviewRepository) Create(ctx context.Context, rv *entity.Review) error {
	const q = `
		INSERT INTO reviews (id, order_item_id, product_id, user_id, rating, comment)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING created_at
	`
	err := r.db.QueryRow(ctx, q, rv.ID, rv.OrderItemID, rv.ProductID, rv.UserID, rv.Rating, rv.Comment).Scan(&rv.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return errs.ErrConflict
		}
		return fmt.Errorf("postgres: create review: %w", err)
	}
	return nil
}

func (r *ReviewRepository) ExistsByOrderItemID(ctx context.Context, orderItemID uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM reviews WHERE order_item_id = $1)`, orderItemID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("postgres: check review exists: %w", err)
	}
	return exists, nil
}

func (r *ReviewRepository) ListByProduct(ctx context.Context, productID uuid.UUID, rating int, cursor string, limit int) ([]*entity.Review, string, error) {
	if limit <= 0 {
		limit = 20
	}

	args := []any{productID}
	where := "product_id = $1"
	if rating > 0 {
		args = append(args, rating)
		where += fmt.Sprintf(" AND rating = $%d", len(args))
	}
	if cursor != "" {
		c, err := decodeCursor(cursor)
		if err != nil {
			return nil, "", fmt.Errorf("postgres: decode cursor: %w", err)
		}
		args = append(args, c.V, c.ID)
		where += fmt.Sprintf(" AND (created_at, id) < ($%d::timestamptz, $%d)", len(args)-1, len(args))
	}
	args = append(args, limit+1)

	q := selectReviewCols + fmt.Sprintf(`FROM reviews WHERE %s ORDER BY created_at DESC, id DESC LIMIT $%d`, where, len(args))
	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, "", fmt.Errorf("postgres: list reviews by product: %w", err)
	}
	defer rows.Close()

	out, err := collectReviews(rows)
	if err != nil {
		return nil, "", err
	}

	var next string
	if len(out) > limit {
		out = out[:limit]
		last := out[limit-1]
		next = encodeCursor(last.CreatedAt.Format(rfc3339Cursor), last.ID)
	}
	return out, next, nil
}

func (r *ReviewRepository) ListByUser(ctx context.Context, userID uuid.UUID, page, perPage int) ([]*entity.Review, int, error) {
	var total int
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM reviews WHERE user_id = $1`, userID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("postgres: count reviews by user: %w", err)
	}

	const q = selectReviewCols + `FROM reviews WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	rows, err := r.db.Query(ctx, q, userID, perPage, (page-1)*perPage)
	if err != nil {
		return nil, 0, fmt.Errorf("postgres: list reviews by user: %w", err)
	}
	defer rows.Close()

	out, err := collectReviews(rows)
	return out, total, err
}

func (r *ReviewRepository) AverageRating(ctx context.Context, productID uuid.UUID) (float64, int, error) {
	const q = `SELECT COALESCE(AVG(rating), 0), COUNT(*) FROM reviews WHERE product_id = $1`
	var avg float64
	var count int
	if err := r.db.QueryRow(ctx, q, productID).Scan(&avg, &count); err != nil {
		return 0, 0, fmt.Errorf("postgres: average rating: %w", err)
	}
	return avg, count, nil
}

const rfc3339Cursor = "2006-01-02T15:04:05.999999999Z07:00"

const selectReviewCols = `SELECT id, order_item_id, product_id, user_id, rating, comment, created_at `

func scanReview(row rowScanner) (*entity.Review, error) {
	var rv entity.Review
	err := row.Scan(&rv.ID, &rv.OrderItemID, &rv.ProductID, &rv.UserID, &rv.Rating, &rv.Comment, &rv.CreatedAt)
	return &rv, err
}

func collectReviews(rows pgx.Rows) ([]*entity.Review, error) {
	var out []*entity.Review
	for rows.Next() {
		rv, err := scanReview(rows)
		if err != nil {
			return nil, fmt.Errorf("postgres: scan review: %w", err)
		}
		out = append(out, rv)
	}
	return out, rows.Err()
}
