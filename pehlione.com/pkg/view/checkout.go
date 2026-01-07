package view

type ShippingOption struct {
	Code  string
	Label string
	Price string
}

type CheckoutForm struct {
	Email      string
	FirstName  string
	LastName   string
	Address1   string
	Address2   string
	City       string
	PostalCode string
	Country    string
	Phone      string

	// Billing address fields
	UseBillingAddress bool
	BillingFirstName  string
	BillingLastName   string
	BillingAddress1   string
	BillingAddress2   string
	BillingCity       string
	BillingPostalCode string
	BillingCountry    string
	BillingPhone      string

	ShippingMethod string
	PaymentMethod  string
	IdemKey        string
}

type CheckoutSummary struct {
	Currency             string
	BaseCurrency         string
	Subtotal             string
	Shipping             string
	Total                string
	Items                int
	Lines                []CartItem
	SubtotalCents        int
	ShippingCents        int
	TotalCents           int
	BaseSubtotalCents    int
	BaseShippingCents    int
	BaseTotalCents       int
	DisplaySubtotalCents int
	DisplayShippingCents int
	DisplayTotalCents    int
}

type PaymentOption struct {
	Code        string
	Label       string
	Description string
}

func ShippingLabel(code string) string {
	switch code {
	case "express":
		return "Express (1-2 gün)"
	case "standard":
		return "Standard (2-4 gün)"
	default:
		if code == "" {
			return "Standard (2-4 gün)"
		}
		return code
	}
}

func PaymentMethodLabel(code string) string {
	switch code {
	case "paypal":
		return "PayPal"
	case "klarna":
		return "Klarna \"Pay Later\""
	case "card", "":
		return "Kart (Visa / Mastercard)"
	default:
		return code
	}
}
