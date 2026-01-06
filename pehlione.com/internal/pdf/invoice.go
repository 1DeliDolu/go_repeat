package pdf

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"

	"pehlione.com/app/internal/modules/orders"
	"pehlione.com/app/pkg/view"
)

type InvoiceData struct {
	Order          orders.Order
	Items          []orders.OrderItem
	ShippingLines  []string
	ShippingMethod string
	PaymentMethod  string
}

const (
	brandYellowR = 250
	brandYellowG = 204
	brandYellowB = 21

	brandOrangeR = 249
	brandOrangeG = 115
	brandOrangeB = 22
)

func GenerateInvoice(data InvoiceData) ([]byte, error) {
	p := fpdf.New("P", "mm", "A4", "")
	p.SetMargins(15, 20, 15)
	p.AddPage()

	renderHeader(p)
	renderOrderMeta(p, data)
	renderItemsTable(p, data.Items)
	renderTotals(p, data.Order)

	var buf bytes.Buffer
	if err := p.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func renderHeader(p *fpdf.Fpdf) {
	p.SetFont("Helvetica", "B", 28)
	p.SetTextColor(brandYellowR, brandYellowG, brandYellowB)
	widthYellow := p.GetStringWidth("pehli")
	p.CellFormat(widthYellow, 12, "pehli", "", 0, "", false, 0, "")
	p.SetTextColor(brandOrangeR, brandOrangeG, brandOrangeB)
	widthOrange := p.GetStringWidth("ONE")
	p.CellFormat(widthOrange, 12, "ONE", "", 1, "", false, 0, "")
	p.Ln(2)
	p.SetTextColor(0, 0, 0)
}

func renderOrderMeta(p *fpdf.Fpdf, data InvoiceData) {
	p.SetFont("Helvetica", "", 11)
	p.CellFormat(40, 6, "Sipariş No:", "", 0, "L", false, 0, "")
	p.CellFormat(0, 6, data.Order.ID, "", 1, "L", false, 0, "")

	p.CellFormat(40, 6, "Tarih:", "", 0, "L", false, 0, "")
	p.CellFormat(0, 6, data.Order.CreatedAt.Local().Format("02.01.2006 15:04"), "", 1, "L", false, 0, "")

	p.CellFormat(40, 6, "Ödeme:", "", 0, "L", false, 0, "")
	p.CellFormat(0, 6, data.PaymentMethod, "", 1, "L", false, 0, "")

	p.CellFormat(40, 6, "Teslimat:", "", 0, "L", false, 0, "")
	p.CellFormat(0, 6, data.ShippingMethod, "", 1, "L", false, 0, "")

	if len(data.ShippingLines) > 0 {
		p.Ln(2)
		p.SetFont("Helvetica", "B", 11)
		p.CellFormat(0, 6, "Teslimat Adresi", "", 1, "", false, 0, "")
		p.SetFont("Helvetica", "", 11)
		for _, line := range data.ShippingLines {
			p.CellFormat(0, 6, line, "", 1, "", false, 0, "")
		}
	}

	p.Ln(4)
}

func renderItemsTable(p *fpdf.Fpdf, items []orders.OrderItem) {
	p.SetFont("Helvetica", "B", 11)
	p.SetFillColor(248, 250, 252)
	p.CellFormat(100, 8, "Ürün", "1", 0, "L", true, 0, "")
	p.CellFormat(30, 8, "Adet", "1", 0, "C", true, 0, "")
	p.CellFormat(50, 8, "Tutar", "1", 1, "R", true, 0, "")

	p.SetFont("Helvetica", "", 11)
	for _, it := range items {
		p.CellFormat(100, 8, truncate(it.ProductName, 60), "1", 0, "L", false, 0, "")
		p.CellFormat(30, 8, fmt.Sprintf("%d", it.Quantity), "1", 0, "C", false, 0, "")
		p.CellFormat(50, 8, view.MoneyFromCents(it.LineTotalCents, it.Currency), "1", 1, "R", false, 0, "")
	}

	p.Ln(4)
}

func renderTotals(p *fpdf.Fpdf, order orders.Order) {
	p.SetFont("Helvetica", "", 11)
	p.CellFormat(130, 6, "", "", 0, "", false, 0, "")
	p.CellFormat(30, 6, "Ara Toplam:", "", 0, "R", false, 0, "")
	p.CellFormat(30, 6, view.MoneyFromCents(order.SubtotalCents, order.Currency), "", 1, "R", false, 0, "")

	p.CellFormat(130, 6, "", "", 0, "", false, 0, "")
	p.CellFormat(30, 6, "Kargo:", "", 0, "R", false, 0, "")
	p.CellFormat(30, 6, view.MoneyFromCents(order.ShippingCents, order.Currency), "", 1, "R", false, 0, "")

	if order.TaxCents > 0 {
		p.CellFormat(130, 6, "", "", 0, "", false, 0, "")
		p.CellFormat(30, 6, "KDV:", "", 0, "R", false, 0, "")
		p.CellFormat(30, 6, view.MoneyFromCents(order.TaxCents, order.Currency), "", 1, "R", false, 0, "")
	}

	p.SetFont("Helvetica", "B", 12)
	p.CellFormat(130, 8, "", "", 0, "", false, 0, "")
	p.CellFormat(30, 8, "Toplam:", "", 0, "R", false, 0, "")
	p.CellFormat(30, 8, view.MoneyFromCents(order.TotalCents, order.Currency), "", 1, "R", false, 0, "")

	p.Ln(8)
	p.SetFont("Helvetica", "", 10)
	p.SetTextColor(120, 124, 139)
	p.CellFormat(0, 5, "pehliONE - Bu belge dijital olarak oluşturuldu ve imza gerektirmez.", "", 1, "C", false, 0, "")
	p.CellFormat(0, 5, time.Now().Format("02.01.2006 15:04"), "", 1, "C", false, 0, "")
}

func truncate(s string, max int) string {
	if len([]rune(s)) <= max {
		return s
	}
	return strings.TrimSpace(string([]rune(s)[:max-1])) + "…"
}
