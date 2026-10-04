package httpserver

import (
	"errors"
	"strconv"

	"github.com/thomas666-beast/marketplace/internal/catalog"
)

func parseLimit(s string) (int, error) {
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 || n > 100 {
		return 0, errors.New("invalid limit")
	}
	return n, nil
}

type categoryResponse struct {
    ID        string             `json:"id"`
    ParentID  *string            `json:"parent_id"`
    Slug      string             `json:"slug"`
    Name      string             `json:"name"`
    SortOrder int                `json:"sort_order"`
    Children  []categoryResponse `json:"children,omitempty"`
}

func toCategoryResponse(c catalog.Category, locale string) categoryResponse {
    return categoryResponse{
        ID:        c.ID,
        ParentID:  c.ParentID,
        Slug:      c.Slug,
        Name:      c.Name.For(locale),
        SortOrder: c.SortOrder,
    }
}

func toCategoryTree(nodes []catalog.CategoryNode, locale string) []categoryResponse {
    out := make([]categoryResponse, 0, len(nodes))
    for _, n := range nodes {
        r := toCategoryResponse(n.Category, locale)
        r.Children = toCategoryTree(n.Children, locale)
        out = append(out, r)
    }
    return out
}

type productResponse struct {
    ID            string `json:"id"`
    SellerID      string `json:"seller_id"`
    CategoryID    string `json:"category_id"`
    Slug          string `json:"slug"`
    Name          string `json:"name"`
    Description   string `json:"description"`
    PriceCents    int64  `json:"price_cents"`
    Currency      string `json:"currency"`
    StockQuantity int    `json:"stock_quantity"`
    Status        string `json:"status"`
    CreatedAt     string `json:"created_at"`
    UpdatedAt     string `json:"updated_at"`
}

func toProductResponse(p catalog.Product) productResponse {
    return productResponse{
        ID:            p.ID,
        SellerID:      p.SellerID,
        CategoryID:    p.CategoryID,
        Slug:          p.Slug,
        Name:          p.Name,
        Description:   p.Description,
        PriceCents:    p.PriceCents,
        Currency:      string(p.Currency),
        StockQuantity: p.StockQuantity,
        Status:        string(p.Status),
        CreatedAt:     p.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
        UpdatedAt:     p.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
    }
}

type productListResponse struct {
    Items      []productResponse `json:"items"`
    NextCursor string            `json:"next_cursor,omitempty"`
}

type createProductRequest struct {
    CategoryID    string `json:"category_id"`
    Name          string `json:"name"`
    Description   string `json:"description"`
    PriceCents    int64  `json:"price_cents"`
    Currency      string `json:"currency"`
    StockQuantity int    `json:"stock_quantity"`
    Status        string `json:"status"`
}

type updateProductRequest struct {
    Name          *string `json:"name"`
    Description   *string `json:"description"`
    PriceCents    *int64  `json:"price_cents"`
    Currency      *string `json:"currency"`
    StockQuantity *int    `json:"stock_quantity"`
    Status        *string `json:"status"`
}
