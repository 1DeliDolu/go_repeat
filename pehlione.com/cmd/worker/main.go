package main

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"pehlione.com/app/internal/config"
	"pehlione.com/app/internal/modules/email"
	"pehlione.com/app/internal/modules/fx"
	"pehlione.com/app/internal/modules/shipping"
	"pehlione.com/app/internal/sms"
)

func main() {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("worker: failed to load config: %v", err)
	}

	db, err := gorm.Open(mysql.Open(cfg.DBDSN), &gorm.Config{})
	if err != nil {
		log.Fatalf("worker: failed to connect database: %v", err)
	}

	var emailSvc *email.OutboxService
	if cfg.Email.Enabled {
		emailSvc = email.NewService(db)
	}

	fxRepo := fx.NewRepo(db)
	fxSvc := fx.NewService(fxRepo, cfg.Currency.BaseCurrency)
	ctx := context.Background()
	errCh := make(chan error, 4)
	started := 0

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	if cfg.Email.Enabled {
		brandName := strings.TrimSpace(cfg.Email.FromName)
		if brandName == "" {
			brandName = "pehliONE"
		}
		supportEmail := strings.TrimSpace(cfg.Email.SMTP.From)
		if supportEmail == "" {
			supportEmail = "support@pehlione.com"
		}
		renderer := email.NewRenderer(cfg.AppBaseURL, brandName, supportEmail)
		sender := email.NewSMTPSender(email.SMTPCfg{
			Host:   cfg.Email.SMTP.Host,
			Port:   cfg.Email.SMTP.Port,
			User:   cfg.Email.SMTP.User,
			Pass:   cfg.Email.SMTP.Pass,
			From:   cfg.Email.SMTP.From,
			UseTLS: cfg.Email.SMTP.UseTLS,
		})

		emailWorker := email.NewWorker(db, sender, renderer)
		started++
		log.Println("email worker starting")
		go func() {
			errCh <- emailWorker.Run(ctx)
		}()
	} else {
		log.Println("worker: email sending disabled")
	}

	if cfg.Shipping.Enabled {
		var shipProvider shipping.Provider
		switch cfg.Shipping.Provider {
		case "", "mock":
			shipProvider = shipping.NewMockProvider(cfg.Shipping.MockBaseURL)
		default:
			log.Fatalf("worker: unsupported shipping provider %s", cfg.Shipping.Provider)
		}

		shipSvc := shipping.NewService(db, shipProvider, emailSvc, cfg.AppBaseURL)
		shipWorker := shipping.NewWorker(shipSvc)
		started++
		log.Println("shipping worker starting")
		go func() {
			errCh <- shipWorker.Run(ctx)
		}()
	} else {
		log.Println("worker: shipping disabled")
	}

	if cfg.SMS.Enabled {
		var smsProvider sms.SMSProvider
		switch cfg.SMS.Provider {
		case "mock":
			smsProvider = sms.NewMockProvider(logger)
		default:
			log.Fatalf("worker: unsupported sms provider %s", cfg.SMS.Provider)
		}

		smsWorker := sms.NewWorker(smsProvider, sms.NewOutboxRepository(db), logger, "sms-worker-1")
		started++
		log.Println("sms worker starting")
		go func() {
			smsWorker.Run(ctx)
		}()
	} else {
		log.Println("worker: sms sending disabled")
	}

	if cfg.Currency.FX.Provider != "" {
		// Use configured symbols, fallback to display currencies if not configured
		symbols := cfg.Currency.FX.Symbols
		if len(symbols) == 0 {
			// Use display currencies except the base currency
			for _, curr := range cfg.Currency.DisplayCurrencies {
				if curr != cfg.Currency.BaseCurrency {
					symbols = append(symbols, curr)
				}
			}
		}

		// Only start FX worker if there are symbols to fetch
		if len(symbols) > 0 {
			var rateProvider fx.Provider
			switch cfg.Currency.FX.Provider {
			case "exchange_rate_host", "exchangerate_host":
				rateProvider = fx.NewExchangeRateHostProvider(symbols)
			default:
				log.Printf("worker: unsupported FX provider %s", cfg.Currency.FX.Provider)
			}
			if rateProvider != nil {
				started++
				interval := time.Duration(cfg.Currency.FX.RefreshMinutes) * time.Minute
				fxWorker := fx.NewWorker(fxSvc, rateProvider, cfg.Currency.BaseCurrency, symbols, interval)
				log.Println("fx worker starting")
				go func() {
					errCh <- fxWorker.Run(ctx)
				}()
			}
		} else {
			log.Println("worker: FX provider configured but no currencies to fetch (single currency mode)")
		}
	}

	if started == 0 {
		log.Println("worker: no workers enabled, exiting")
		return
	}

	err = <-errCh
	if err != nil && !errors.Is(err, context.Canceled) {
		log.Fatalf("worker stopped: %v", err)
	}
}
