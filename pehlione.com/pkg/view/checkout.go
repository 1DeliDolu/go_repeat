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

	ShippingMethod string
	IdemKey        string
}

type CheckoutSummary struct {
	Currency string
	Subtotal string
	Shipping string
	Total    string
	Items    int
}
