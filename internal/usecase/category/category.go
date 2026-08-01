// Package category implements admin CRUD and the public category tree
// (FR-4.1), cached for 10 minutes and invalidated on any mutation.
package category

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"erdinhrmwn/bangunin/internal/domain/entity"
	"erdinhrmwn/bangunin/internal/domain/errs"
	"erdinhrmwn/bangunin/internal/domain/repository"
	"erdinhrmwn/bangunin/pkg/apperr"
	"erdinhrmwn/bangunin/pkg/cache"
)

const (
	treeCacheKey = "category:tree"
	treeCacheTTL = 10 * time.Minute
	maxDepth     = 3
)

type Usecase struct {
	categories repository.CategoryRepository
	rdb        *redis.Client
}

func New(categories repository.CategoryRepository, rdb *redis.Client) *Usecase {
	return &Usecase{categories: categories, rdb: rdb}
}

type Input struct {
	ParentID  *int
	Name      string
	IsActive  bool
	SortOrder int
}

func (u *Usecase) Create(ctx context.Context, in Input) (*entity.Category, error) {
	if err := u.validateParent(ctx, in.ParentID); err != nil {
		return nil, err
	}
	slug, err := u.uniqueSlug(ctx, in.Name, 0)
	if err != nil {
		return nil, err
	}
	c := &entity.Category{
		ParentID:  in.ParentID,
		Name:      in.Name,
		Slug:      slug,
		IsActive:  in.IsActive,
		SortOrder: in.SortOrder,
	}
	if err := u.categories.Create(ctx, c); err != nil {
		return nil, err
	}
	u.invalidateTree(ctx)
	return c, nil
}

func (u *Usecase) Update(ctx context.Context, id int, in Input) (*entity.Category, error) {
	c, err := u.categories.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if in.ParentID != nil && *in.ParentID == id {
		return nil, apperr.New("INVALID_PARENT", "A category cannot be its own parent", 422)
	}
	if err := u.validateParent(ctx, in.ParentID); err != nil {
		return nil, err
	}

	if in.Name != c.Name {
		slug, err := u.uniqueSlug(ctx, in.Name, id)
		if err != nil {
			return nil, err
		}
		c.Slug = slug
	}
	c.ParentID = in.ParentID
	c.Name = in.Name
	c.IsActive = in.IsActive
	c.SortOrder = in.SortOrder

	if err := u.categories.Update(ctx, c); err != nil {
		return nil, err
	}
	u.invalidateTree(ctx)
	return c, nil
}

// Delete removes a category, refusing when it still has children or
// products attached (FR-4.1).
func (u *Usecase) Delete(ctx context.Context, id int) error {
	if _, err := u.categories.FindByID(ctx, id); err != nil {
		return err
	}
	children, err := u.categories.CountChildren(ctx, id)
	if err != nil {
		return err
	}
	if children > 0 {
		return apperr.New("HAS_CHILDREN", "Category has child categories", 409)
	}
	products, err := u.categories.CountProducts(ctx, id)
	if err != nil {
		return err
	}
	if products > 0 {
		return apperr.New("HAS_PRODUCTS", "Category has products", 409)
	}
	if err := u.categories.Delete(ctx, id); err != nil {
		return err
	}
	u.invalidateTree(ctx)
	return nil
}

// Tree returns the full category tree, cache-aside for 10 minutes.
func (u *Usecase) Tree(ctx context.Context) ([]*entity.Category, error) {
	if cached, ok, err := cache.GetJSON[[]*entity.Category](ctx, u.rdb, treeCacheKey); err == nil && ok {
		return cached, nil
	}
	list, err := u.categories.ListTree(ctx)
	if err != nil {
		return nil, err
	}
	_ = cache.SetJSON(ctx, u.rdb, treeCacheKey, list, treeCacheTTL)
	return list, nil
}

func (u *Usecase) invalidateTree(ctx context.Context) {
	_ = cache.Delete(ctx, u.rdb, treeCacheKey)
}

// validateParent ensures the parent exists and the resulting depth doesn't
// exceed maxDepth (FR-4.1: max 3 levels).
func (u *Usecase) validateParent(ctx context.Context, parentID *int) error {
	if parentID == nil {
		return nil
	}
	depth := 1
	id := *parentID
	for {
		p, err := u.categories.FindByID(ctx, id)
		if errors.Is(err, errs.ErrNotFound) {
			return apperr.New("PARENT_NOT_FOUND", "Parent category not found", 422)
		}
		if err != nil {
			return err
		}
		depth++
		if depth > maxDepth {
			return apperr.New("MAX_DEPTH", "Category hierarchy limited to 3 levels", 422)
		}
		if p.ParentID == nil {
			return nil
		}
		id = *p.ParentID
	}
}

var slugNonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(name string) string {
	s := slugNonAlnum.ReplaceAllString(strings.ToLower(name), "-")
	return strings.Trim(s, "-")
}

// uniqueSlug slugifies name and appends a short suffix on collision,
// excluding the category identified by selfID (0 means "new category").
func (u *Usecase) uniqueSlug(ctx context.Context, name string, selfID int) (string, error) {
	base := slugify(name)
	slug := base
	for i := 2; ; i++ {
		existing, err := u.categories.FindBySlug(ctx, slug)
		if errors.Is(err, errs.ErrNotFound) {
			return slug, nil
		}
		if err != nil {
			return "", err
		}
		if existing.ID == selfID {
			return slug, nil
		}
		slug = fmt.Sprintf("%s-%d", base, i)
	}
}
