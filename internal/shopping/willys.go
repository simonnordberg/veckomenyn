package shopping

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/simonnordberg/veckomenyn/internal/providers"
	"github.com/simonnordberg/veckomenyn/internal/willys"
)

// WillysProvider adapts the existing Willys HTTP client to the Provider
// interface, handles lazy login from DB-stored credentials, and retries
// once on auth failure.
type WillysProvider struct {
	client    *willys.Client
	providers *providers.Store
	log       *slog.Logger

	mu    sync.Mutex
	creds providers.WillysCreds // last seen creds; re-login on change
}

func NewWillys(pool *pgxpool.Pool, prov *providers.Store, log *slog.Logger) *WillysProvider {
	return &WillysProvider{
		client:    willys.NewClientWithStore(NewDBSessionStore(pool)),
		providers: prov,
		log:       log,
	}
}

func (p *WillysProvider) Kind() string { return "willys" }

// ensureLoggedIn logs in when:
//   - we have no cookies yet, or
//   - the credentials configured in Settings have changed since last login.
//
// Serialised to avoid double-login across concurrent requests.
func (p *WillysProvider) ensureLoggedIn(ctx context.Context, force bool) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	creds, ok := p.providers.WillysCredentials(ctx)
	if !ok {
		return errors.New("willys provider is not configured (Settings → Integrations)")
	}

	credsChanged := creds != p.creds
	if !force && !credsChanged && p.client.IsLoggedIn() {
		return nil
	}

	if credsChanged {
		p.client.ClearState()
	}
	if _, err := p.client.Login(creds.Username, creds.Password); err != nil {
		return fmt.Errorf("willys login: %w", err)
	}
	p.creds = creds
	return nil
}

// withAuth runs fn, retrying once after forcing a fresh login if the first
// call looks like an auth failure.
func (p *WillysProvider) withAuth(ctx context.Context, fn func() error) error {
	if err := p.ensureLoggedIn(ctx, false); err != nil {
		return err
	}
	err := fn()
	if err == nil {
		return nil
	}
	if !looksLikeAuthError(err) {
		return err
	}
	p.log.Info("willys auth retry")
	if err := p.ensureLoggedIn(ctx, true); err != nil {
		return err
	}
	return fn()
}

func looksLikeAuthError(err error) bool {
	if err == nil {
		return false
	}
	if willys.IsAuthError(err) {
		return true
	}
	// Defensive substring match for the existing client's text-error surface.
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "401") || strings.Contains(s, "unauthorized") ||
		strings.Contains(s, "403") || strings.Contains(s, "forbidden")
}

// ---------------------------------------------------------------------------
// Provider methods
// ---------------------------------------------------------------------------

func (p *WillysProvider) Search(ctx context.Context, query string, limit int) ([]Product, error) {
	if limit <= 0 {
		limit = 10
	}
	var out []Product
	err := p.withAuth(ctx, func() error {
		// Willys.se paginates from page 0. Passing 1 skips the first page,
		// which for small queries (2 results total) returns an empty set.
		// The CLI's search works the same way.
		res, err := p.client.Search(query, 0, limit)
		if err != nil {
			return err
		}
		out = make([]Product, 0, limit)
		for _, r := range res.Results {
			out = append(out, toProduct(r))
		}
		// If the query has no direct hits, Willys returns results=null and
		// puts category-adjacent products in relatedResults. Fill from there
		// so the agent gets something usable to work with.
		for _, r := range res.RelatedResults {
			if len(out) >= limit {
				break
			}
			pr := toProduct(r)
			pr.Related = true
			out = append(out, pr)
		}
		return nil
	})
	return out, err
}

func (p *WillysProvider) CartGet(ctx context.Context) (Cart, error) {
	var out Cart
	err := p.withAuth(ctx, func() error {
		c, err := p.client.GetCart()
		if err != nil {
			return err
		}
		out = toCart(c)
		return nil
	})
	return out, err
}

func (p *WillysProvider) CartAdd(ctx context.Context, code string, qty int) (Cart, error) {
	if qty <= 0 {
		qty = 1
	}
	var out Cart
	err := p.withAuth(ctx, func() error {
		c, err := p.client.AddToCart(code, qty)
		if err != nil {
			return err
		}
		out = toCart(c)
		return nil
	})
	return out, err
}

func (p *WillysProvider) CartRemove(ctx context.Context, code string) (Cart, error) {
	var out Cart
	err := p.withAuth(ctx, func() error {
		c, err := p.client.RemoveFromCart(code)
		if err != nil {
			return err
		}
		out = toCart(c)
		return nil
	})
	return out, err
}

func (p *WillysProvider) CartClear(ctx context.Context) error {
	return p.withAuth(ctx, func() error { return p.client.ClearCart() })
}

func (p *WillysProvider) OrdersRecent(ctx context.Context, limit int) ([]OrderSummary, error) {
	if limit <= 0 {
		limit = 5
	}
	var out []OrderSummary
	err := p.withAuth(ctx, func() error {
		list, err := p.client.GetOrderHistory()
		if err != nil {
			return err
		}
		if limit > len(list) {
			limit = len(list)
		}
		out = make([]OrderSummary, 0, limit)
		for _, o := range list[:limit] {
			// Willys returns `total` as a formatted string ("1 234,50 kr")
			// and no nested totalPrice, so fall back through both.
			total := o.TotalPrice.Value
			if total == 0 {
				total = parseKronor(o.Total)
			}
			out = append(out, OrderSummary{
				ID:     firstNonEmpty(o.OrderNumber, o.Code),
				Date:   firstNonEmpty(o.DeliveryDate, o.OrderDate),
				Total:  total,
				Status: o.OrderStatus.Code,
			})
		}
		return nil
	})
	return out, err
}

func (p *WillysProvider) OrderDetail(ctx context.Context, id string) (Order, error) {
	var out Order
	err := p.withAuth(ctx, func() error {
		d, err := p.client.GetOrderDetail(id)
		if err != nil {
			return err
		}
		total := d.TotalPrice.Value
		if total == 0 {
			total = parseKronor(d.TotalPrice.FormattedValue)
		}
		out = Order{
			ID:     firstNonEmpty(d.OrderNumber, d.Code),
			Date:   d.DeliveryDate,
			Total:  total,
			Status: firstNonEmpty(d.StatusDisplay, d.OrderStatus.Code),
		}
		for category, entries := range d.Products {
			for _, e := range entries {
				out.Lines = append(out.Lines, OrderLine{
					Code:      e.Code,
					Name:      e.Name,
					Qty:       float64(e.Quantity),
					LineTotal: parseKronor(e.TotalPrice),
					Category:  category,
				})
			}
		}
		return nil
	})
	return out, err
}

// ---------------------------------------------------------------------------
// willys → neutral type translation
// ---------------------------------------------------------------------------

func toProduct(p willys.Product) Product {
	return Product{
		Code:         p.Code,
		Name:         p.Name,
		Price:        p.PriceValue,
		PricePerUnit: strings.TrimSpace(p.ComparePrice + " " + p.ComparePriceUnit),
		Unit:         p.DisplayVolume,
		OnPromotion:  len(p.PotentialPromotions) > 0,
	}
}

func toCart(c willys.Cart) Cart {
	out := Cart{Total: parseKronor(c.TotalPrice)}
	for _, l := range c.Products {
		out.Items = append(out.Items, CartLine{
			Code:      l.Code,
			Name:      l.Name,
			Qty:       float64(l.Quantity),
			UnitPrice: l.PriceValue,
			LineTotal: parseKronor(l.TotalPrice),
		})
	}
	return out
}

// parseKronor extracts a numeric value from strings like "1.234,50 kr" that
// the Willys API returns in its "TotalPrice" fields. Returns 0 on miss.
func parseKronor(s string) float64 {
	// Strip everything that isn't a digit, comma, or dot.
	cleaned := strings.Map(func(r rune) rune {
		if (r >= '0' && r <= '9') || r == ',' || r == '.' {
			return r
		}
		return -1
	}, s)
	// Swedish format uses comma as decimal separator and . as thousands.
	cleaned = strings.ReplaceAll(cleaned, ".", "")
	cleaned = strings.ReplaceAll(cleaned, ",", ".")
	if cleaned == "" {
		return 0
	}
	v, err := strconv.ParseFloat(cleaned, 64)
	if err != nil {
		return 0
	}
	return v
}

func firstNonEmpty(s ...string) string {
	for _, v := range s {
		if v != "" {
			return v
		}
	}
	return ""
}
