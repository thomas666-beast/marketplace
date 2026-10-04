package catalog

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CategoryRepository struct {
	pool *pgxpool.Pool
}

func NewCategoryRepository(pool *pgxpool.Pool) *CategoryRepository {
	return &CategoryRepository{pool: pool}
}

const categoryColumns = `
	id, parent_id, slug, name_en, name_ru, name_es,
	sort_order, is_active, created_at, updated_at
`

func scanCategory(row pgx.Row) (Category, error) {
	var c Category
	err := row.Scan(
		&c.ID,
		&c.ParentID,
		&c.Slug,
		&c.Name.EN,
		&c.Name.RU,
		&c.Name.ES,
		&c.SortOrder,
		&c.IsActive,
		&c.CreatedAt,
		&c.UpdatedAt,
	)
	return c, err
}

func (r *CategoryRepository) ListActive(ctx context.Context) ([]Category, error) {
	const q = `
		SELECT ` + categoryColumns + `
		FROM categories
		WHERE is_active = TRUE
		ORDER BY parent_id NULLS FIRST, sort_order, slug
	`

	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	defer rows.Close()

	var categories []Category
	for rows.Next() {
		c, err := scanCategory(rows)
		if err != nil {
			return nil, fmt.Errorf("scan category: %w", err)
		}
		categories = append(categories, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate categories: %w", err)
	}
	return categories, nil
}

func (r *CategoryRepository) TreeActive(ctx context.Context) ([]CategoryNode, error) {
	flat, err := r.ListActive(ctx)
	if err != nil {
		return nil, err
	}
	return buildTree(flat), nil
}

// buildTree assembles a flat list of categories into a tree.
// Children whose parent is not present in the input are treated as roots.
func buildTree(flat []Category) []CategoryNode {
	present := make(map[string]bool, len(flat))
	for _, c := range flat {
		present[c.ID] = true
	}

	childrenOf := make(map[string][]Category)
	var roots []Category

	for _, c := range flat {
		if c.ParentID == nil || !present[*c.ParentID] {
			roots = append(roots, c)
			continue
		}
		childrenOf[*c.ParentID] = append(childrenOf[*c.ParentID], c)
	}

	var build func(c Category) CategoryNode
	build = func(c Category) CategoryNode {
		kids := childrenOf[c.ID]
		sort.SliceStable(kids, func(i, j int) bool {
			return kids[i].SortOrder < kids[j].SortOrder
		})
		node := CategoryNode{
			Category: c,
			Children: make([]CategoryNode, 0, len(kids)),
		}
		for _, k := range kids {
			node.Children = append(node.Children, build(k))
		}
		return node
	}

	sort.SliceStable(roots, func(i, j int) bool {
		return roots[i].SortOrder < roots[j].SortOrder
	})

	out := make([]CategoryNode, 0, len(roots))
	for _, r := range roots {
		out = append(out, build(r))
	}
	return out
}

func (r *CategoryRepository) GetByID(ctx context.Context, id string) (Category, error) {
	const q = `SELECT ` + categoryColumns + ` FROM categories WHERE id = $1 AND is_active = TRUE`

	c, err := scanCategory(r.pool.QueryRow(ctx, q, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Category{}, ErrCategoryNotFound
		}
		return Category{}, fmt.Errorf("get category: %w", err)
	}
	return c, nil
}

func (r *CategoryRepository) GetBySlug(ctx context.Context, slug string) (Category, error) {
	const q = `SELECT ` + categoryColumns + ` FROM categories WHERE slug = $1 AND is_active = TRUE`

	c, err := scanCategory(r.pool.QueryRow(ctx, q, slug))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Category{}, ErrCategoryNotFound
		}
		return Category{}, fmt.Errorf("get category by slug: %w", err)
	}
	return c, nil
}

func (r *CategoryRepository) Deactivate(ctx context.Context, id string) error {
	const q = `UPDATE categories SET is_active = FALSE, updated_at = NOW()
	           WHERE id = $1 AND is_active = TRUE`

	tag, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return ErrCategoryInUse
		}
		return fmt.Errorf("deactivate category: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrCategoryNotFound
	}
	return nil
}
