package willys

// Customer represents a logged-in Willys user.
type Customer struct {
	UID       string `json:"uid"`
	Name      string `json:"name"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Email     string `json:"email"`
	StoreID   string `json:"storeId"`
}

// Promotion represents a potential deal on a product.
type Promotion struct {
	ConditionLabel  string         `json:"conditionLabel"`
	RewardLabel     string         `json:"rewardLabel"`
	Price           FormattedPrice `json:"price"`
	QualifyingCount int            `json:"qualifyingCount"`
	PromotionType   string         `json:"promotionType"`
}

// Product represents a grocery product.
type Product struct {
	Name                string      `json:"name"`
	Code                string      `json:"code"`
	Price               string      `json:"price"`
	PriceValue          float64     `json:"priceValue"`
	ComparePrice        string      `json:"comparePrice"`
	ComparePriceUnit    string      `json:"comparePriceUnit"`
	Manufacturer        string      `json:"manufacturer"`
	DisplayVolume       string      `json:"displayVolume"`
	OutOfStock          bool        `json:"outOfStock"`
	PotentialPromotions []Promotion `json:"potentialPromotions"`
	SavingsAmount       *float64    `json:"savingsAmount"`
}

// CartProduct is a product in the shopping cart.
type CartProduct struct {
	Code             string  `json:"code"`
	Name             string  `json:"name"`
	Price            string  `json:"price"`
	PriceValue       float64 `json:"priceValue"`
	Quantity         int     `json:"quantity"`
	PickQuantity     int     `json:"pickQuantity"`
	TotalPrice       string  `json:"totalPrice"`
	Manufacturer     string  `json:"manufacturer"`
	DisplayVolume    string  `json:"displayVolume"`
	ComparePrice     string  `json:"comparePrice"`
	ComparePriceUnit string  `json:"comparePriceUnit"`
}

// Cart represents the shopping cart.
type Cart struct {
	Products       []CartProduct `json:"products"`
	TotalPrice     string        `json:"totalPrice"`
	TotalItems     int           `json:"totalItems"`
	TotalUnitCount int           `json:"totalUnitCount"`
	TotalDiscount  string        `json:"totalDiscount"`
}

// Pagination holds paging info for search/browse results.
type Pagination struct {
	PageSize             int `json:"pageSize"`
	CurrentPage          int `json:"currentPage"`
	NumberOfPages        int `json:"numberOfPages"`
	TotalNumberOfResults int `json:"totalNumberOfResults"`
}

// SearchResult is the response from product search or category browse.
//
// Willys splits hits into two buckets: Results are direct matches for the
// query; RelatedResults are broader category / fuzzy matches. When there
// is no exact hit, Results comes back nil and everything useful is in
// RelatedResults. Callers that want "did we find something I can cook
// with?" should consider both.
type SearchResult struct {
	Results                  []Product  `json:"results"`
	Pagination               Pagination `json:"pagination"`
	RelatedResults           []Product  `json:"relatedResults"`
	RelatedResultsPagination Pagination `json:"relatedResultsPagination"`
}

// FormattedPrice is a price object returned by the API with value and display string.
type FormattedPrice struct {
	Value          float64 `json:"value"`
	FormattedValue string  `json:"formattedValue"`
	CurrencyISO    string  `json:"currencyIso"`
}

// OrderStatus wraps the status code returned by the API.
type OrderStatus struct {
	Code string `json:"code"`
}

// OrderSummary is a single entry in the order history list.
type OrderSummary struct {
	OrderNumber    string         `json:"orderNumber"`
	Code           string         `json:"code"`
	OrderStatus    OrderStatus    `json:"orderStatus"`
	Completed      bool           `json:"completed"`
	OrderDate      string         `json:"formattedOrderDate"`
	DeliveryDate   string         `json:"deliveryFormattedDate"`
	Total          string         `json:"total"`
	TotalPrice     FormattedPrice `json:"totalPrice"`
	ReservedAmount string         `json:"reservedAmount"`
}

// OrderEntry is a line item in an order.
type OrderEntry struct {
	Code          string  `json:"code"`
	Name          string  `json:"name"`
	Quantity      int     `json:"quantity"`
	PickQuantity  int     `json:"pickQuantity"`
	Price         string  `json:"price"`
	PriceValue    float64 `json:"priceValue"`
	TotalPrice    string  `json:"totalPrice"`
	Manufacturer  string  `json:"manufacturer"`
	DisplayVolume string  `json:"displayVolume"`
	CategoryName  string  `json:"categoryName"`
}

// OrderDetail is the full detail of a single order.
type OrderDetail struct {
	OrderNumber   string                  `json:"orderNumber"`
	Code          string                  `json:"code"`
	OrderStatus   OrderStatus             `json:"orderStatus"`
	StatusDisplay string                  `json:"statusDisplay"`
	Completed     bool                    `json:"completed"`
	DeliveryDate  string                  `json:"deliveryFormattedDate"`
	TotalPrice    FormattedPrice          `json:"totalPrice"`
	NettoCost     FormattedPrice          `json:"nettoTotalCost"`
	Products      map[string][]OrderEntry `json:"categoryOrderedDeliveredProducts"`
}

// Category represents a node in the category tree.
type Category struct {
	ID       string     `json:"id"`
	Title    string     `json:"title"`
	URL      string     `json:"url"`
	Children []Category `json:"children"`
}
