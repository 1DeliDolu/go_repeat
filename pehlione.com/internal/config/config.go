package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type SMTPConfig struct {
	Host          string
	Port          string
	User          string
	Pass          string
	From          string
	TLSMode       string // none|starttls|tls
	SkipVerifyTLS bool
}

type MailtrapConfig struct {
	APIURL   string
	APIToken string
}

type EmailConfig struct {
	Enabled  bool
	From     string
	FromName string
	SMTP     SMTPConfig
	Mailtrap MailtrapConfig
}

type PaymentConfig struct {
	Provider             string
	APIKey               string
	PublicKey            string
	WebhookSecret        string
	LiveMode             bool
	MockWebhookSecret    string
	MockWebhookTolerance int
}

type FXConfig struct {
	Provider       string
	APIKey         string
	Symbols        []string
	RefreshMinutes int
}

type CurrencyConfig struct {
	BaseCurrency      string
	DefaultDisplay    string
	DisplayCurrencies []string
	ChargeCurrencies  []string
	CookieName        string
	CookieSecure      bool
	FX                FXConfig
}

type AppConfig struct {
	Env        string
	DBDSN      string
	AppBaseURL string
	Email      EmailConfig
	Payment    PaymentConfig
	Shipping   ShippingConfig
	SMS        SMSConfig
	Currency   CurrencyConfig
}

func Load() (AppConfig, error) {
	cfg := AppConfig{
		Env:        strings.ToLower(strings.TrimSpace(getEnv("APP_ENV", "development"))),
		DBDSN:      strings.TrimSpace(os.Getenv("DB_DSN")),
		AppBaseURL: strings.TrimSpace(getEnv("APP_BASE_URL", "http://localhost:8080")),
	}
	if cfg.DBDSN == "" {
		return AppConfig{}, fmt.Errorf("DB_DSN is required")
	}

	cfg.Email = loadEmailConfig()
	cfg.Payment = loadPaymentConfig()
	cfg.Shipping = loadShippingConfig()
	cfg.SMS = loadSMSConfig()
	cfg.Currency = loadCurrencyConfig()

	if err := validateConfig(&cfg); err != nil {
		return AppConfig{}, err
	}

	return cfg, nil
}

func loadEmailConfig() EmailConfig {
	smtp := SMTPConfig{
		Host:          strings.TrimSpace(getEnv("SMTP_HOST", "localhost")),
		Port:          strings.TrimSpace(getEnv("SMTP_PORT", "1025")),
		User:          strings.TrimSpace(os.Getenv("SMTP_USER")),
		Pass:          strings.TrimSpace(os.Getenv("SMTP_PASS")),
		From:          strings.TrimSpace(getEnv("SMTP_FROM", "no-reply@pehlione.com")),
		TLSMode:       strings.ToLower(strings.TrimSpace(getEnv("SMTP_TLS_MODE", "none"))),
		SkipVerifyTLS: parseBool(getEnv("SMTP_SKIP_VERIFY_TLS", "false"), false),
	}

	emailFrom := strings.TrimSpace(os.Getenv("EMAIL_FROM"))
	if emailFrom == "" {
		emailFrom = smtp.From
	}

	return EmailConfig{
		Enabled:  parseBool(getEnv("EMAIL_SEND_ENABLED", "true"), true),
		From:     emailFrom,
		FromName: strings.TrimSpace(getEnv("EMAIL_FROM_NAME", "Pehlione")),
		SMTP:     smtp,
		Mailtrap: MailtrapConfig{
			APIURL:   strings.TrimSpace(os.Getenv("MAILTRAP_API_URL")),
			APIToken: strings.TrimSpace(os.Getenv("MAILTRAP_API_TOKEN")),
		},
	}
}

func loadPaymentConfig() PaymentConfig {
	return PaymentConfig{
		Provider:             strings.ToLower(strings.TrimSpace(getEnv("PAYMENT_PROVIDER", "mock"))),
		APIKey:               strings.TrimSpace(os.Getenv("PAYMENT_API_KEY")),
		PublicKey:            strings.TrimSpace(os.Getenv("PAYMENT_PUBLIC_KEY")),
		WebhookSecret:        strings.TrimSpace(os.Getenv("PAYMENT_WEBHOOK_SECRET")),
		LiveMode:             parseBool(getEnv("PAYMENT_LIVE_MODE", "false"), false),
		MockWebhookSecret:    strings.TrimSpace(getEnv("MOCK_WEBHOOK_SECRET", "dev_secret_change_me")),
		MockWebhookTolerance: parseInt(getEnv("MOCK_WEBHOOK_TOLERANCE_SECONDS", "300"), 300),
	}
}

type ShippingConfig struct {
	Enabled     bool
	Provider    string
	MockBaseURL string
}

type SMSConfig struct {
	Enabled  bool
	Provider string
}

func loadShippingConfig() ShippingConfig {
	return ShippingConfig{
		Enabled:     parseBool(getEnv("SHIPPING_ENABLED", "true"), true),
		Provider:    strings.ToLower(strings.TrimSpace(getEnv("SHIPPING_PROVIDER", "mock"))),
		MockBaseURL: strings.TrimSpace(os.Getenv("SHIPPING_MOCK_BASE_URL")),
	}
}

func loadSMSConfig() SMSConfig {
	return SMSConfig{
		Enabled:  parseBool(getEnv("SMS_ENABLED", "true"), true),
		Provider: strings.ToLower(strings.TrimSpace(getEnv("SMS_PROVIDER", "mock"))),
	}
}

func loadCurrencyConfig() CurrencyConfig {
	base := strings.ToUpper(strings.TrimSpace(getEnv("CURRENCY_BASE", "TRY")))
	defaultDisplay := strings.ToUpper(strings.TrimSpace(getEnv("CURRENCY_DEFAULT_DISPLAY", base)))
	display := parseCSV(getEnv("CURRENCY_ALLOWED", base))
	if len(display) == 0 {
		display = []string{base}
	}
	charge := parseCSV(getEnv("CURRENCY_CHARGE", base))
	if len(charge) == 0 {
		charge = []string{base}
	}

	return CurrencyConfig{
		BaseCurrency:      base,
		DefaultDisplay:    defaultDisplay,
		DisplayCurrencies: display,
		ChargeCurrencies:  charge,
		CookieName:        strings.TrimSpace(getEnv("CURRENCY_COOKIE_NAME", "currency_pref")),
		CookieSecure:      parseBool(getEnv("CURRENCY_COOKIE_SECURE", "false"), false),
		FX: FXConfig{
			Provider:       strings.ToLower(strings.TrimSpace(getEnv("FX_PROVIDER", "exchange_rate_host"))),
			APIKey:         strings.TrimSpace(os.Getenv("FX_API_KEY")),
			Symbols:        parseCSV(getEnv("FX_SYMBOLS", "")),
			RefreshMinutes: parseInt(getEnv("FX_REFRESH_MINUTES", "180"), 180),
		},
	}
}

func validateConfig(cfg *AppConfig) error {
	if cfg.Email.Enabled {
		if cfg.Email.SMTP.Host == "" {
			return fmt.Errorf("SMTP_HOST is required when email sending is enabled")
		}
		if cfg.Email.SMTP.Port == "" {
			return fmt.Errorf("SMTP_PORT is required when email sending is enabled")
		}
		// Validate Port is a valid number
		if _, err := strconv.Atoi(cfg.Email.SMTP.Port); err != nil {
			return fmt.Errorf("SMTP_PORT must be a valid port number: %w", err)
		}
		if cfg.Email.SMTP.From == "" {
			return fmt.Errorf("SMTP_FROM is required when email sending is enabled")
		}
	}

	if cfg.Payment.MockWebhookTolerance <= 0 {
		cfg.Payment.MockWebhookTolerance = 300
	}

	if cfg.Env == "production" {
		if cfg.Payment.Provider == "mock" {
			return fmt.Errorf("mock payment provider is not allowed in production")
		}
		if !cfg.Payment.LiveMode {
			return fmt.Errorf("PAYMENT_LIVE_MODE must be true in production")
		}
		if cfg.Payment.APIKey == "" || cfg.Payment.PublicKey == "" || cfg.Payment.WebhookSecret == "" {
			return fmt.Errorf("payment credentials are required in production")
		}
	}

	if cfg.Shipping.Enabled {
		switch cfg.Shipping.Provider {
		case "", "mock":
		default:
			return fmt.Errorf("unsupported shipping provider: %s", cfg.Shipping.Provider)
		}
	}

	if cfg.Currency.BaseCurrency == "" {
		return fmt.Errorf("CURRENCY_BASE is required")
	}
	if cfg.Currency.DefaultDisplay == "" {
		cfg.Currency.DefaultDisplay = cfg.Currency.BaseCurrency
	}
	if len(cfg.Currency.DisplayCurrencies) == 0 {
		cfg.Currency.DisplayCurrencies = []string{cfg.Currency.BaseCurrency}
	}
	if cfg.Currency.FX.RefreshMinutes <= 0 {
		cfg.Currency.FX.RefreshMinutes = 180
	}

	return nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func parseBool(val string, def bool) bool {
	if strings.TrimSpace(val) == "" {
		return def
	}
	b, err := strconv.ParseBool(val)
	if err != nil {
		return def
	}
	return b
}

func parseInt(val string, def int) int {
	if strings.TrimSpace(val) == "" {
		return def
	}
	n, err := strconv.Atoi(val)
	if err != nil {
		return def
	}
	return n
}

func parseCSV(val string) []string {
	val = strings.TrimSpace(val)
	if val == "" {
		return nil
	}
	parts := strings.Split(val, ",")
	out := make([]string, 0, len(parts))
	seen := make(map[string]struct{})
	for _, part := range parts {
		code := strings.ToUpper(strings.TrimSpace(part))
		if code == "" {
			continue
		}
		if _, ok := seen[code]; ok {
			continue
		}
		seen[code] = struct{}{}
		out = append(out, code)
	}
	return out
}
