package shared

import "fmt"

// FormatMoney para miktarını biçimlendirir. cents cinsinden alır ve para birimini ekler.
func FormatMoney(currency string, cents int64) string {
	major := float64(cents) / 100.0
	switch currency {
	case "EUR":
		return fmt.Sprintf("€%.2f", major)
	case "TRY":
		return fmt.Sprintf("₺%.2f", major)
	case "USD":
		return fmt.Sprintf("$%.2f", major)
	default:
		return fmt.Sprintf("%.2f %s", major, currency)
	}
}
