package entity

import "time"

type Category struct {
	ID        int
	ParentID  *int
	Name      string
	Slug      string
	IsActive  bool
	SortOrder int
	CreatedAt time.Time
}
