package view

import "fmt"

// MoneyFromCents converts cents to a human-readable currency string.
// E.g., 1000 EUR -> "€10.00"
func MoneyFromCents(cents int, currency string) string {
	euros := cents / 100
	remainder := cents % 100
	return fmt.Sprintf("%s%.2f", currencySymbol(currency), float64(euros)+float64(remainder)/100)
}

func currencySymbol(code string) string {
	switch code {
	case "EUR":
		return "€"
	case "USD":
		return "$"
	case "GBP":
		return "£"
	case "JPY":
		return "¥"
	case "TRY":
		return "₺"
	default:
		return code + " "
	}
}
