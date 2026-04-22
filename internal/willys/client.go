package willys

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
)

const (
	baseURL        = "https://www.willys.se"
	defaultStoreID = "2110"
)

// Client talks to the Willys.se API.
//
// Safe for concurrent use. Mutable session state (cookies, CSRF token) is
// protected by mu; the underlying *http.Client is already safe.
type Client struct {
	http         *http.Client
	mu           sync.RWMutex      // protects cookies, csrfToken
	cookies      map[string]string // guarded by mu
	csrfToken    string            // guarded by mu
	baseOverride string            // test-only, immutable after construction
	store        SessionStore      // itself safe for concurrent use
}

// NewClient creates a new Willys API client backed by the default file
// session store (CLI behaviour, preserved for backwards compatibility).
func NewClient() *Client {
	return NewClientWithStore(DefaultFileStore())
}

// NewClientWithStore creates a client whose session is loaded/saved via the
// given store. Server code passes a DB-backed store so session state lives
// alongside the rest of the app data.
func NewClientWithStore(store SessionStore) *Client {
	c := &Client{
		http: &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		cookies: make(map[string]string),
		store:   store,
	}
	c.loadSession()
	return c
}

func (c *Client) loadSession() {
	if c.store == nil {
		return
	}
	s, err := c.store.Load(context.Background())
	if err != nil {
		return
	}
	c.mu.Lock()
	if s.Cookies != nil {
		c.cookies = s.Cookies
	}
	c.csrfToken = s.CSRFToken
	c.mu.Unlock()
}

func (c *Client) saveSession() {
	if c.store == nil {
		return
	}
	c.mu.RLock()
	cookies := make(map[string]string, len(c.cookies))
	for k, v := range c.cookies {
		cookies[k] = v
	}
	csrf := c.csrfToken
	c.mu.RUnlock()
	_ = c.store.Save(context.Background(), Session{
		Cookies:   cookies,
		CSRFToken: csrf,
	})
}

// ClearSession removes the saved session via the default file store. Kept
// for CLI backwards compatibility; server code should call store.Clear
// directly on its own store.
func ClearSession() {
	_ = DefaultFileStore().Clear(context.Background())
}

// needsCSRF reports whether we need to fetch a fresh CSRF token before the
// next mutating request. Safe to call under concurrent access.
func (c *Client) needsCSRF() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.csrfToken == ""
}

// ClearState resets the in-memory session and erases it from the store.
// Useful when the server needs to force a re-login.
func (c *Client) ClearState() {
	c.mu.Lock()
	c.cookies = map[string]string{}
	c.csrfToken = ""
	c.mu.Unlock()
	if c.store != nil {
		_ = c.store.Clear(context.Background())
	}
}

// IsAuthError reports whether an error came from a 401/403 response.
func IsAuthError(err error) bool {
	var ae *authErr
	return errors.As(err, &ae)
}

type authErr struct {
	status int
}

func (e *authErr) Error() string {
	return fmt.Sprintf("auth failed: %d", e.status)
}

func (c *Client) base() string {
	if c.baseOverride != "" {
		return c.baseOverride
	}
	return baseURL
}

func (c *Client) do(method, path string, body any) (*http.Response, error) {
	u := path
	if !strings.HasPrefix(path, "http") {
		u = c.base() + path
	}

	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, u, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Snapshot session state under the lock, then release before I/O so we
	// don't block concurrent callers during an HTTP round trip.
	c.mu.RLock()
	csrf := c.csrfToken
	var cookieHeader string
	if len(c.cookies) > 0 {
		parts := make([]string, 0, len(c.cookies))
		for k, v := range c.cookies {
			parts = append(parts, k+"="+v)
		}
		cookieHeader = strings.Join(parts, "; ")
	}
	c.mu.RUnlock()

	if method != http.MethodGet && csrf != "" {
		req.Header.Set("X-CSRF-TOKEN", csrf)
	}
	// Go's cookiejar is too strict for Willys' flow — attach manually.
	if cookieHeader != "" {
		req.Header.Set("Cookie", cookieHeader)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	c.parseCookies(resp)
	return resp, nil
}

func (c *Client) parseCookies(resp *http.Response) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, header := range resp.Header.Values("Set-Cookie") {
		nameVal := strings.SplitN(header, ";", 2)[0]
		if eqIdx := strings.Index(nameVal, "="); eqIdx > 0 {
			name := strings.TrimSpace(nameVal[:eqIdx])
			value := strings.TrimSpace(nameVal[eqIdx+1:])
			c.cookies[name] = value
		}
	}
}

func (c *Client) fetchCSRFToken() error {
	resp, err := c.do(http.MethodGet, "/axfood/rest/csrf-token", nil)
	if err != nil {
		return fmt.Errorf("fetching CSRF token: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("CSRF token request failed: %d", resp.StatusCode)
	}
	var token string
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return fmt.Errorf("decoding CSRF token: %w", err)
	}
	c.mu.Lock()
	c.csrfToken = token
	c.mu.Unlock()
	return nil
}

func (c *Client) ensureSession() error {
	c.mu.RLock()
	_, hasSession := c.cookies["JSESSIONID"]
	c.mu.RUnlock()
	if hasSession {
		return nil
	}
	resp, err := c.do(http.MethodGet, "/api/config", nil)
	if err != nil {
		return fmt.Errorf("establishing session: %w", err)
	}
	_ = resp.Body.Close()
	return nil
}

// Login authenticates with the Willys API using encrypted credentials.
func (c *Client) Login(username, password string) (Customer, error) {
	if err := c.ensureSession(); err != nil {
		return Customer{}, err
	}
	if err := c.fetchCSRFToken(); err != nil {
		return Customer{}, err
	}

	encUser, err := EncryptCredential(username)
	if err != nil {
		return Customer{}, fmt.Errorf("encrypting username: %w", err)
	}
	encPass, err := EncryptCredential(password)
	if err != nil {
		return Customer{}, fmt.Errorf("encrypting password: %w", err)
	}

	loginBody := map[string]any{
		"j_username":     encUser.Str,
		"j_username_key": encUser.Key,
		"j_password":     encPass.Str,
		"j_password_key": encPass.Key,
		"j_remember_me":  true,
	}

	resp, err := c.do(http.MethodPost, "/login", loginBody)
	if err != nil {
		return Customer{}, fmt.Errorf("login request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 200))
		return Customer{}, fmt.Errorf("login failed (%d): %s", resp.StatusCode, body)
	}

	if loc := resp.Header.Get("Location"); loc != "" {
		r, err := c.do(http.MethodGet, loc, nil)
		if err != nil {
			return Customer{}, fmt.Errorf("following login redirect: %w", err)
		}
		_ = r.Body.Close()
	}

	if err := c.fetchCSRFToken(); err != nil {
		return Customer{}, err
	}

	cust, err := c.GetCustomer()
	if err != nil {
		return Customer{}, err
	}

	c.saveSession()
	return cust, nil
}

// IsLoggedIn checks if the saved session is still valid.
func (c *Client) IsLoggedIn() bool {
	c.mu.RLock()
	hasCookies := len(c.cookies) > 0
	c.mu.RUnlock()
	if !hasCookies {
		return false
	}
	cust, err := c.GetCustomer()
	if err != nil {
		return false
	}
	return cust.FirstName != "" && cust.Name != "anonymous"
}

// GetCustomer returns the logged-in user's profile.
func (c *Client) GetCustomer() (Customer, error) {
	resp, err := c.do(http.MethodGet, "/axfood/rest/customer", nil)
	if err != nil {
		return Customer{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Customer{}, fmt.Errorf("get customer failed: %d", resp.StatusCode)
	}
	var cust Customer
	return cust, json.NewDecoder(resp.Body).Decode(&cust)
}

// Search finds products by query.
func (c *Client) Search(query string, page, size int) (SearchResult, error) {
	params := url.Values{
		"q":    {query},
		"size": {strconv.Itoa(size)},
		"page": {strconv.Itoa(page)},
	}
	resp, err := c.do(http.MethodGet, "/search/clean?"+params.Encode(), nil)
	if err != nil {
		return SearchResult{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return SearchResult{}, fmt.Errorf("search failed: %d", resp.StatusCode)
	}
	var result SearchResult
	return result, json.NewDecoder(resp.Body).Decode(&result)
}

// GetProduct fetches full product details including promotions.
func (c *Client) GetProduct(code string) (Product, error) {
	resp, err := c.do(http.MethodGet, "/axfood/rest/p/"+code, nil)
	if err != nil {
		return Product{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Product{}, fmt.Errorf("get product failed: %d", resp.StatusCode)
	}
	var p Product
	return p, json.NewDecoder(resp.Body).Decode(&p)
}

// Categories returns the full category tree.
func (c *Client) Categories() (Category, error) {
	params := url.Values{
		"storeId":    {defaultStoreID},
		"deviceType": {"OTHER"},
	}
	resp, err := c.do(http.MethodGet, "/leftMenu/categorytree?"+params.Encode(), nil)
	if err != nil {
		return Category{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Category{}, fmt.Errorf("categories failed: %d", resp.StatusCode)
	}
	var cat Category
	return cat, json.NewDecoder(resp.Body).Decode(&cat)
}

// Browse lists products in a category.
func (c *Client) Browse(categoryPath string, page, size int) (SearchResult, error) {
	params := url.Values{
		"page": {strconv.Itoa(page)},
		"size": {strconv.Itoa(size)},
		"sort": {""},
	}
	resp, err := c.do(http.MethodGet, "/c/"+categoryPath+"?"+params.Encode(), nil)
	if err != nil {
		return SearchResult{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return SearchResult{}, fmt.Errorf("browse failed: %d", resp.StatusCode)
	}
	var result SearchResult
	return result, json.NewDecoder(resp.Body).Decode(&result)
}

// GetCart returns the current shopping cart.
func (c *Client) GetCart() (Cart, error) {
	resp, err := c.do(http.MethodGet, "/axfood/rest/cart", nil)
	if err != nil {
		return Cart{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Cart{}, fmt.Errorf("get cart failed: %d", resp.StatusCode)
	}
	var cart Cart
	return cart, json.NewDecoder(resp.Body).Decode(&cart)
}

type addProductEntry struct {
	ProductCodePost     string `json:"productCodePost"`
	Qty                 int    `json:"qty"`
	PickUnit            string `json:"pickUnit"`
	HideDiscountToolTip bool   `json:"hideDiscountToolTip"`
	NoReplacementFlag   bool   `json:"noReplacementFlag"`
}

// AddToCart adds products to the cart and returns the updated cart.
func (c *Client) AddToCart(code string, qty int) (Cart, error) {
	if c.needsCSRF() {
		if err := c.fetchCSRFToken(); err != nil {
			return Cart{}, err
		}
	}
	body := map[string][]addProductEntry{
		"products": {
			{
				ProductCodePost: code,
				Qty:             qty,
				PickUnit:        "pieces",
			},
		},
	}
	resp, err := c.do(http.MethodPost, "/axfood/rest/cart/addProducts", body)
	if err != nil {
		return Cart{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 200))
		return Cart{}, fmt.Errorf("add to cart failed (%d): %s", resp.StatusCode, b)
	}
	return c.GetCart()
}

// RemoveFromCart removes a product from the cart.
func (c *Client) RemoveFromCart(code string) (Cart, error) {
	return c.AddToCart(code, 0)
}

// GetOrderHistory returns the order history for the logged-in user.
// The API may return either a raw array or an object with an "orders" key.
func (c *Client) GetOrderHistory() ([]OrderSummary, error) {
	resp, err := c.do(http.MethodGet, "/axfood/rest/account/orders", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get order history failed: %d", resp.StatusCode)
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	// Try decoding as array first, then as object with "orders" key.
	var orders []OrderSummary
	if err := json.Unmarshal(raw, &orders); err == nil {
		return orders, nil
	}
	var wrapper struct {
		Orders []OrderSummary `json:"orders"`
	}
	if err := json.Unmarshal(raw, &wrapper); err != nil {
		return nil, fmt.Errorf("decoding order history: %w", err)
	}
	return wrapper.Orders, nil
}

// GetOrderDetail returns the full details of a single order.
func (c *Client) GetOrderDetail(orderNumber string) (OrderDetail, error) {
	params := url.Values{
		"q": {orderNumber},
	}
	resp, err := c.do(http.MethodGet, "/axfood/rest/orderdata?"+params.Encode(), nil)
	if err != nil {
		return OrderDetail{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return OrderDetail{}, fmt.Errorf("get order detail failed: %d", resp.StatusCode)
	}
	var order OrderDetail
	return order, json.NewDecoder(resp.Body).Decode(&order)
}

// ClearCart removes all products from the cart.
func (c *Client) ClearCart() error {
	if c.needsCSRF() {
		if err := c.fetchCSRFToken(); err != nil {
			return err
		}
	}
	resp, err := c.do(http.MethodDelete, "/axfood/rest/cart", nil)
	if err != nil {
		return err
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("clear cart failed: %d", resp.StatusCode)
	}
	return nil
}
