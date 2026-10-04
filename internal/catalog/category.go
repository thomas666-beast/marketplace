package catalog

import "time"

type LocalizedName struct {
	EN string
	RU string
	ES string
}

func (n LocalizedName) For(locale string) string {
	switch locale {
	case "ru":
		return n.RU
	case "es":
		return n.ES
	default:
		return n.EN
	}
}

type Category struct {
	ID         string
	ParentID   *string
	Slug       string
	Name       LocalizedName
	SortOrder  int
	IsActive   bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// CategoryNode is a Category with its children attached.
type CategoryNode struct {
	Category
	Children []CategoryNode
}
