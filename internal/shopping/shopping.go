// Package shopping defines a provider-neutral shopping API (search / cart /
// orders) and the wiring to instantiate the configured provider (Willys
// today, ICA or others in the future).
package shopping

import "context"

// Product is the minimal fields the agent cares about: a code to reference
// in the cart, a name, the last-seen price, and optionally a unit label.
type Product struct {
	Code         string  `json:"code"`
	Name         string  `json:"name"`
	Price        float64 `json:"price"`          // kronor
	PricePerUnit string  `json:"price_per_unit"` // e.g. "14,60 kr/l"
	Unit         string  `json:"unit"`           // e.g. "ST" or "KG"
	OnPromotion  bool    `json:"on_promotion,omitempty"`
	// Related is true when the product came from the provider's fuzzy /
	// related-category fallback rather than an exact-query hit.
	Related bool `json:"related,omitempty"`
}

type CartLine struct {
	Code      string  `json:"code"`
	Name      string  `json:"name"`
	Qty       float64 `json:"qty"`
	UnitPrice float64 `json:"unit_price"`
	LineTotal float64 `json:"line_total"`
}

type Cart struct {
	Items []CartLine `json:"items"`
	Total float64    `json:"total"`
}

type OrderSummary struct {
	ID     string  `json:"id"`
	Date   string  `json:"date"`
	Total  float64 `json:"total"`
	Status string  `json:"status,omitempty"`
}

type OrderLine struct {
	Code      string  `json:"code"`
	Name      string  `json:"name"`
	Qty       float64 `json:"qty"`
	LineTotal float64 `json:"line_total"`
	Category  string  `json:"category,omitempty"`
}

type Order struct {
	ID     string      `json:"id"`
	Date   string      `json:"date"`
	Total  float64     `json:"total"`
	Status string      `json:"status"`
	Lines  []OrderLine `json:"lines"`
}

// Provider is the interface every shopping integration implements. The kind
// string matches the providers table row (e.g. "willys").
type Provider interface {
	Kind() string
	Search(ctx context.Context, query string, limit int) ([]Product, error)
	CartGet(ctx context.Context) (Cart, error)
	CartAdd(ctx context.Context, code string, qty int) (Cart, error)
	CartRemove(ctx context.Context, code string) (Cart, error)
	CartClear(ctx context.Context) error
	OrdersRecent(ctx context.Context, limit int) ([]OrderSummary, error)
	OrderDetail(ctx context.Context, id string) (Order, error)
}
