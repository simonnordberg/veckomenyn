package willys

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestParseOrderHistory_Array(t *testing.T) {
	raw := `[{"orderNumber":"3057837654","formattedOrderDate":"2026-03-24","orderStatus":{"code":"delivered"},"total":"3 291,29 kr"}]`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(raw))
	}))
	defer srv.Close()

	c := &Client{http: srv.Client(), cookies: map[string]string{}}
	c.baseOverride = srv.URL
	orders, err := c.GetOrderHistory()
	if err != nil {
		t.Fatalf("GetOrderHistory: %v", err)
	}
	if len(orders) != 1 {
		t.Fatalf("got %d orders, want 1", len(orders))
	}
	if orders[0].OrderNumber != "3057837654" {
		t.Errorf("order number = %q, want %q", orders[0].OrderNumber, "3057837654")
	}
}

func TestParseOrderHistory_Wrapper(t *testing.T) {
	raw := `{"orders":[{"orderNumber":"3057837654","formattedOrderDate":"2026-03-24","orderStatus":{"code":"delivered"},"total":"3 291,29 kr"}]}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(raw))
	}))
	defer srv.Close()

	c := &Client{http: srv.Client(), cookies: map[string]string{}}
	c.baseOverride = srv.URL
	orders, err := c.GetOrderHistory()
	if err != nil {
		t.Fatalf("GetOrderHistory: %v", err)
	}
	if len(orders) != 1 {
		t.Fatalf("got %d orders, want 1", len(orders))
	}
}

func TestParseOrderHistory_Empty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	c := &Client{http: srv.Client(), cookies: map[string]string{}}
	c.baseOverride = srv.URL
	orders, err := c.GetOrderHistory()
	if err != nil {
		t.Fatalf("GetOrderHistory: %v", err)
	}
	if len(orders) != 0 {
		t.Fatalf("got %d orders, want 0", len(orders))
	}
}

func TestParseSearchResult(t *testing.T) {
	raw := `{
		"results": [{"name":"Mjölk","code":"100010649_ST","price":"21,90 kr","manufacturer":"Falköpings","displayVolume":"1,5l"}],
		"pagination": {"totalNumberOfResults":1,"numberOfPages":1,"currentPage":0,"pageSize":10}
	}`
	var result SearchResult
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(result.Results) != 1 {
		t.Fatalf("got %d results, want 1", len(result.Results))
	}
	p := result.Results[0]
	if p.Code != "100010649_ST" {
		t.Errorf("code = %q", p.Code)
	}
	if p.Manufacturer != "Falköpings" {
		t.Errorf("manufacturer = %q", p.Manufacturer)
	}
	if result.Pagination.TotalNumberOfResults != 1 {
		t.Errorf("total = %d", result.Pagination.TotalNumberOfResults)
	}
}

func TestSearchUsesMultiWordEndpoint(t *testing.T) {
	// Willys has two search endpoints: /search/clean returns a degenerate
	// single-token match (q=färsk koriander → Färskost). /search does proper
	// multi-word relevance. We must hit /search.
	var gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.Query().Get("q")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[],"pagination":{}}`))
	}))
	defer srv.Close()

	c := &Client{http: srv.Client(), cookies: map[string]string{}}
	c.baseOverride = srv.URL
	if _, err := c.Search("färsk koriander", 0, 10); err != nil {
		t.Fatalf("Search: %v", err)
	}
	if gotPath != "/search" {
		t.Errorf("path = %q, want %q", gotPath, "/search")
	}
	if gotQuery != "färsk koriander" {
		t.Errorf("q = %q, want %q", gotQuery, "färsk koriander")
	}
}

func TestParseOrderDetail(t *testing.T) {
	raw := `{
		"orderNumber":"3057837654",
		"statusDisplay":"Levererad",
		"orderStatus":{"code":"delivered"},
		"totalPrice":{"value":3291.29,"formattedValue":"3 291,29 kr","currencyIso":"SEK"},
		"categoryOrderedDeliveredProducts":{
			"Mejeri":[{"name":"Mjölk","code":"100010649_ST","pickQuantity":2,"totalPrice":"43,80 kr","manufacturer":"Falköpings","displayVolume":"1,5l"}]
		}
	}`
	var order OrderDetail
	if err := json.Unmarshal([]byte(raw), &order); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if order.OrderNumber != "3057837654" {
		t.Errorf("order number = %q", order.OrderNumber)
	}
	if order.StatusDisplay != "Levererad" {
		t.Errorf("status = %q", order.StatusDisplay)
	}
	items, ok := order.Products["Mejeri"]
	if !ok || len(items) != 1 {
		t.Fatalf("Mejeri items = %d", len(items))
	}
	if items[0].PickQuantity != 2 {
		t.Errorf("pick qty = %d", items[0].PickQuantity)
	}
}

func TestParseCart(t *testing.T) {
	raw := `{
		"products":[{"code":"101476110_ST","name":"A-fil 3%","pickQuantity":4,"totalPrice":"63,60 kr","manufacturer":"Garant","displayVolume":"1kg"}],
		"totalPrice":"63,60 kr",
		"totalItems":1,
		"totalUnitCount":4
	}`
	var cart Cart
	if err := json.Unmarshal([]byte(raw), &cart); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if cart.TotalUnitCount != 4 {
		t.Errorf("unit count = %d", cart.TotalUnitCount)
	}
	if len(cart.Products) != 1 {
		t.Fatalf("products = %d", len(cart.Products))
	}
	if cart.Products[0].Code != "101476110_ST" {
		t.Errorf("code = %q", cart.Products[0].Code)
	}
}

func TestParseProductWithPromotions(t *testing.T) {
	raw := `{
		"results": [{
			"name":"Prosciutto Crudo Skivad",
			"code":"101206348_ST",
			"price":"37,76 kr",
			"priceValue":37.76,
			"manufacturer":"Garant",
			"displayVolume":"80g",
			"comparePrice":"472,00 kr",
			"comparePriceUnit":"kg",
			"savingsAmount":25.52,
			"potentialPromotions":[{
				"conditionLabel":"2 för",
				"rewardLabel":"50,00",
				"qualifyingCount":2,
				"promotionType":"MixMatchPricePromotion",
				"price":{"value":25.0,"formattedValue":"25,00 kr","currencyIso":"SEK"}
			}]
		}],
		"pagination":{"totalNumberOfResults":1,"numberOfPages":1,"currentPage":0,"pageSize":10}
	}`
	var result SearchResult
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	p := result.Results[0]
	if p.SavingsAmount == nil || *p.SavingsAmount != 25.52 {
		t.Errorf("savingsAmount = %v, want 25.52", p.SavingsAmount)
	}
	if len(p.PotentialPromotions) != 1 {
		t.Fatalf("promotions = %d, want 1", len(p.PotentialPromotions))
	}
	promo := p.PotentialPromotions[0]
	if promo.ConditionLabel != "2 för" {
		t.Errorf("conditionLabel = %q", promo.ConditionLabel)
	}
	if promo.RewardLabel != "50,00" {
		t.Errorf("rewardLabel = %q", promo.RewardLabel)
	}
	if promo.QualifyingCount != 2 {
		t.Errorf("qualifyingCount = %d", promo.QualifyingCount)
	}
	if promo.Price.Value != 25.0 {
		t.Errorf("price.value = %f", promo.Price.Value)
	}
}

func TestParseProductNoPromotions(t *testing.T) {
	raw := `{
		"results": [{"name":"Mjölk","code":"100010649_ST","price":"21,90 kr","manufacturer":"Falköpings","displayVolume":"1,5l"}],
		"pagination":{"totalNumberOfResults":1,"numberOfPages":1,"currentPage":0,"pageSize":10}
	}`
	var result SearchResult
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	p := result.Results[0]
	if p.SavingsAmount != nil {
		t.Errorf("savingsAmount should be nil, got %v", *p.SavingsAmount)
	}
	if len(p.PotentialPromotions) != 0 {
		t.Errorf("promotions should be empty, got %d", len(p.PotentialPromotions))
	}
}

func TestGetProduct(t *testing.T) {
	raw := `{
		"name":"Prosciutto Crudo Skivad",
		"code":"101206348_ST",
		"price":"37,76 kr",
		"priceValue":37.76,
		"manufacturer":"Garant",
		"displayVolume":"80g",
		"savingsAmount":25.52,
		"potentialPromotions":[{
			"conditionLabel":"2 för",
			"rewardLabel":"50,00",
			"qualifyingCount":2,
			"promotionType":"MixMatchPricePromotion",
			"price":{"value":25.0,"formattedValue":"25,00 kr","currencyIso":"SEK"}
		}]
	}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/axfood/rest/p/101206348_ST" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(raw))
	}))
	defer srv.Close()

	c := &Client{http: srv.Client(), cookies: map[string]string{}}
	c.baseOverride = srv.URL

	p, err := c.GetProduct("101206348_ST")
	if err != nil {
		t.Fatalf("GetProduct: %v", err)
	}
	if p.Code != "101206348_ST" {
		t.Errorf("code = %q", p.Code)
	}
	if p.SavingsAmount == nil || *p.SavingsAmount != 25.52 {
		t.Errorf("savingsAmount = %v", p.SavingsAmount)
	}
	if len(p.PotentialPromotions) != 1 {
		t.Fatalf("promotions = %d", len(p.PotentialPromotions))
	}
	if p.PotentialPromotions[0].ConditionLabel != "2 för" {
		t.Errorf("conditionLabel = %q", p.PotentialPromotions[0].ConditionLabel)
	}
}

func TestSessionRoundTrip(t *testing.T) {
	store := &FileSessionStore{Path: filepath.Join(t.TempDir(), "session.json")}

	c := &Client{
		http:      &http.Client{},
		cookies:   map[string]string{"JSESSIONID": "abc123", "other": "val"},
		csrfToken: "tok-456",
		store:     store,
	}
	c.saveSession()

	c2 := &Client{
		http:    &http.Client{},
		cookies: make(map[string]string),
		store:   store,
	}
	c2.loadSession()

	if c2.cookies["JSESSIONID"] != "abc123" {
		t.Errorf("JSESSIONID = %q", c2.cookies["JSESSIONID"])
	}
	if c2.csrfToken != "tok-456" {
		t.Errorf("csrf = %q", c2.csrfToken)
	}

	c2.ClearState()
	if _, err := os.Stat(store.Path); !os.IsNotExist(err) {
		t.Error("session file should be deleted after ClearState")
	}
}
