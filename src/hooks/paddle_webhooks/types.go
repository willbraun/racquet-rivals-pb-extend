package paddle_webhooks

import "time"

type PaddleDrawEntryTransaction = PaddleTransaction[DrawEntryCustomData]

type PaddleTransaction[T any] struct {
	EventID        string             `json:"event_id"`
	EventType      string             `json:"event_type"`
	OccurredAt     time.Time          `json:"occurred_at"`
	NotificationID string             `json:"notification_id"`
	Data           TransactionData[T] `json:"data"`
}

type TransactionData[T any] struct {
	ID             string             `json:"id"`
	Items          []TransactionItem  `json:"items"`
	Origin         string             `json:"origin"`
	Status         string             `json:"status"`
	Details        TransactionDetails `json:"details"`
	Checkout       Checkout           `json:"checkout"`
	Payments       []Payment          `json:"payments"`
	BilledAt       time.Time          `json:"billed_at"`
	AddressID      string             `json:"address_id"`
	CreatedAt      time.Time          `json:"created_at"`
	InvoiceID      string             `json:"invoice_id"`
	UpdatedAt      time.Time          `json:"updated_at"`
	RevisedAt      *time.Time         `json:"revised_at"`
	BusinessID     *string            `json:"business_id"`
	CustomData     *T                 `json:"custom_data"`
	CustomerID     string             `json:"customer_id"`
	DiscountID     *string            `json:"discount_id"`
	CurrencyCode   string             `json:"currency_code"`
	BillingPeriod  BillingPeriod      `json:"billing_period"`
	InvoiceNumber  string             `json:"invoice_number"`
	BillingDetails any                `json:"billing_details"`
	CollectionMode string             `json:"collection_mode"`
	SubscriptionID string             `json:"subscription_id"`
}

type DrawEntryCustomData struct {
	UserID       string  `json:"user_id"`
	MensDrawID   *string `json:"mens_draw_id,omitempty"`
	WomensDrawID *string `json:"womens_draw_id,omitempty"`
}

type TransactionItem struct {
	Price     Price `json:"price"`
	Quantity  int   `json:"quantity"`
	Proration any   `json:"proration"`
}

type Price struct {
	ID                 string        `json:"id"`
	Name               string        `json:"name"`
	Type               string        `json:"type"`
	Status             string        `json:"status"`
	Quantity           PriceQuantity `json:"quantity"`
	TaxMode            string        `json:"tax_mode"`
	CreatedAt          time.Time     `json:"created_at"`
	ProductID          string        `json:"product_id"`
	UnitPrice          UnitPrice     `json:"unit_price"`
	UpdatedAt          time.Time     `json:"updated_at"`
	CustomData         any           `json:"custom_data"`
	Description        string        `json:"description"`
	TrialPeriod        any           `json:"trial_period"`
	BillingCycle       *BillingCycle `json:"billing_cycle"`
	UnitPriceOverrides []any         `json:"unit_price_overrides"`
	ImportMeta         any           `json:"import_meta"`
}

type PriceQuantity struct {
	Maximum int `json:"maximum"`
	Minimum int `json:"minimum"`
}

type UnitPrice struct {
	Amount       string `json:"amount"`
	CurrencyCode string `json:"currency_code"`
}

type BillingCycle struct {
	Interval  string `json:"interval"`
	Frequency int    `json:"frequency"`
}

type TransactionDetails struct {
	Totals         Totals         `json:"totals"`
	LineItems      []LineItem     `json:"line_items"`
	PayoutTotals   Totals         `json:"payout_totals"`
	TaxRatesUsed   []TaxRateUsed  `json:"tax_rates_used"`
	AdjustedTotals AdjustedTotals `json:"adjusted_totals"`
}

type Totals struct {
	Fee             string `json:"fee"`
	Tax             string `json:"tax"`
	Total           string `json:"total"`
	Credit          string `json:"credit"`
	Balance         string `json:"balance"`
	Discount        string `json:"discount"`
	Earnings        string `json:"earnings"`
	Subtotal        string `json:"subtotal"`
	GrandTotal      string `json:"grand_total"`
	CurrencyCode    string `json:"currency_code"`
	CreditToBalance string `json:"credit_to_balance"`
}

type LineItem struct {
	ID         string     `json:"id"`
	Totals     ItemTotals `json:"totals"`
	Product    Product    `json:"product"`
	PriceID    string     `json:"price_id"`
	Quantity   int        `json:"quantity"`
	TaxRate    string     `json:"tax_rate"`
	UnitTotals ItemTotals `json:"unit_totals"`
}

type ItemTotals struct {
	Tax      string `json:"tax"`
	Total    string `json:"total"`
	Discount string `json:"discount"`
	Subtotal string `json:"subtotal"`
}

type Product struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	Status      string    `json:"status"`
	ImageURL    string    `json:"image_url"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	CustomData  any       `json:"custom_data"`
	Description string    `json:"description"`
	TaxCategory string    `json:"tax_category"`
	ImportMeta  any       `json:"import_meta"`
}

type TaxRateUsed struct {
	Totals  ItemTotals `json:"totals"`
	TaxRate string     `json:"tax_rate"`
}

type AdjustedTotals struct {
	Fee          string `json:"fee"`
	Tax          string `json:"tax"`
	Total        string `json:"total"`
	Earnings     string `json:"earnings"`
	Subtotal     string `json:"subtotal"`
	GrandTotal   string `json:"grand_total"`
	CurrencyCode string `json:"currency_code"`
}

type Checkout struct {
	URL string `json:"url"`
}

type Payment struct {
	Amount                string        `json:"amount"`
	Status                string        `json:"status"`
	CreatedAt             time.Time     `json:"created_at"`
	ErrorCode             *string       `json:"error_code"`
	CapturedAt            *time.Time    `json:"captured_at"`
	MethodDetails         MethodDetails `json:"method_details"`
	PaymentMethodID       string        `json:"payment_method_id"`
	PaymentAttemptID      string        `json:"payment_attempt_id"`
	StoredPaymentMethodID string        `json:"stored_payment_method_id"`
}

type MethodDetails struct {
	Card *CardDetails `json:"card,omitempty"`
	Type string       `json:"type"`
}

type CardDetails struct {
	Type           string `json:"type"`
	Last4          string `json:"last4"`
	ExpiryYear     int    `json:"expiry_year"`
	ExpiryMonth    int    `json:"expiry_month"`
	CardholderName string `json:"cardholder_name"`
}

type BillingPeriod struct {
	EndsAt   time.Time `json:"ends_at"`
	StartsAt time.Time `json:"starts_at"`
}

// Paddle subscription types

type PaddleSubscriptionActivated = PaddleSubscription[SubscriptionActivatedCustomData]

type PaddleSubscription[T any] struct {
	EventID        string              `json:"event_id"`
	EventType      string              `json:"event_type"`
	OccurredAt     time.Time           `json:"occurred_at"`
	NotificationID string              `json:"notification_id"`
	Data           SubscriptionData[T] `json:"data"`
	Meta           RequestMeta         `json:"meta"`
}

type RequestMeta struct {
	RequestID string `json:"request_id"`
}

type SubscriptionData[T any] struct {
	ID                   string             `json:"id"`
	Status               string             `json:"status"`
	CustomerID           string             `json:"customer_id"`
	AddressID            string             `json:"address_id"`
	BusinessID           *string            `json:"business_id"`
	CurrencyCode         string             `json:"currency_code"`
	CreatedAt            time.Time          `json:"created_at"`
	UpdatedAt            time.Time          `json:"updated_at"`
	StartedAt            time.Time          `json:"started_at"`
	FirstBilledAt        time.Time          `json:"first_billed_at"`
	NextBilledAt         time.Time          `json:"next_billed_at"`
	PausedAt             *time.Time         `json:"paused_at"`
	CanceledAt           *time.Time         `json:"canceled_at"`
	CollectionMode       string             `json:"collection_mode"`
	BillingDetails       any                `json:"billing_details"`
	CurrentBillingPeriod BillingPeriod      `json:"current_billing_period"`
	BillingCycle         BillingCycle       `json:"billing_cycle"`
	ScheduledChange      *any               `json:"scheduled_change"`
	Items                []SubscriptionItem `json:"items"`
	CustomData           *T                 `json:"custom_data"`
	ManagementURLs       ManagementURLs     `json:"management_urls"`
	Discount             *any               `json:"discount"`
	ImportMeta           *any               `json:"import_meta"`
}

type SubscriptionActivatedCustomData struct {
	UserID string `json:"user_id"`
}

type SubscriptionItem struct {
	Status             string    `json:"status"`
	Quantity           int       `json:"quantity"`
	Recurring          bool      `json:"recurring"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
	PreviouslyBilledAt time.Time `json:"previously_billed_at"`
	NextBilledAt       time.Time `json:"next_billed_at"`
	TrialDates         *any      `json:"trial_dates"`
	Price              Price     `json:"price"`
	Product            Product   `json:"product"`
}

type ManagementURLs struct {
	UpdatePaymentMethod string `json:"update_payment_method"`
	Cancel              string `json:"cancel"`
}
