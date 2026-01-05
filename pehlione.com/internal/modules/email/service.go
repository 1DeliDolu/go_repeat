package email

type Service interface {
	SendEmail(to string, toName string, subject string, htmlBody string, textBody string) error
}

// Example: Send order confirmation email
func SendOrderConfirmation(svc Service, customerEmail string, customerName string, orderID string, total string) error {
	subject := "Sipariş Onayı - Pehlione"
	textBody := "Merhaba " + customerName + ",\n\nSiparişiniz (#" + orderID + ") alındı. Toplam: " + total + "\n\nTeşekkürler!"

	htmlBody := `
<html>
  <body style="font-family: sans-serif;">
    <h2>Sipariş Onayı</h2>
    <p>Merhaba ` + customerName + `,</p>
    <p>Siparişiniz alındı.</p>
    <p><strong>Sipariş No:</strong> #` + orderID + `</p>
    <p><strong>Toplam:</strong> ` + total + `</p>
    <p>Teşekkürler!</p>
    <p>Pehlione Ekibi</p>
  </body>
</html>
`

	return svc.SendEmail(customerEmail, customerName, subject, htmlBody, textBody)
}

// Example: Send password reset email
func SendPasswordReset(svc Service, email string, resetLink string) error {
	subject := "Şifre Sıfırla - Pehlione"
	textBody := "Şifrenizi sıfırlamak için: " + resetLink

	htmlBody := `
<html>
  <body style="font-family: sans-serif;">
    <h2>Şifre Sıfırla</h2>
    <p><a href="` + resetLink + `">Şifrenizi sıfırlamak için tıklayın</a></p>
    <p>Bu link 1 saat içinde geçerliliğini yitirecektir.</p>
  </body>
</html>
`

	return svc.SendEmail(email, "", subject, htmlBody, textBody)
}

// Example: Send welcome email
func SendWelcome(svc Service, email string, name string) error {
	subject := "Pehlione'ye Hoş Geldiniz!"
	textBody := "Merhaba " + name + ",\n\nPehlione'ye katıldığınız için teşekkürler!"

	htmlBody := `
<html>
  <body style="font-family: sans-serif;">
    <h2>Hoş Geldiniz!</h2>
    <p>Merhaba ` + name + `,</p>
    <p>Pehlione'ye katıldığınız için teşekkürler!</p>
    <p>Kaliteli ürünler keşfetmeye başlayın.</p>
  </body>
</html>
`

	return svc.SendEmail(email, name, subject, htmlBody, textBody)
}
