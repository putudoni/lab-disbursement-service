package server

import (
	"strconv"
	"time"

	"gofr.dev/pkg/gofr"
	gofrservice "gofr.dev/pkg/gofr/service"

	"lab-disbursement-service/internal/job"
	"lab-disbursement-service/internal/repository"
	"lab-disbursement-service/internal/service"
	"lab-disbursement-service/pkg/pg_provider"
	"lab-disbursement-service/pkg/pg_provider/adapters/xendit"
	"lab-disbursement-service/pkg/pg_provider/adapters/xenditmock"
)

func New() *gofr.App {
	app := gofr.New()

	validator := initPGProvider(app)
	db := newDBHolder()

	gormCfg := gormConfig{
		Host:     app.Config.GetOrDefault("DB_HOST", "localhost"),
		Port:     app.Config.GetOrDefault("DB_PORT", "5432"),
		User:     app.Config.GetOrDefault("DB_USER", "postgres"),
		Password: app.Config.GetOrDefault("DB_PASSWORD", ""),
		Name:     app.Config.GetOrDefault("DB_NAME", ""),
		SSLMode:  app.Config.GetOrDefault("DB_SSL_MODE", "disable"),
		TimeZone: app.Config.GetOrDefault("DB_TIMEZONE", "UTC"),
	}

	bankAccountRepo := repository.NewBankAccountRepository(db)
	validationSvc := service.NewBankAccountValidationService(bankAccountRepo, validator)

	app.OnStart(func(ctx *gofr.Context) error {
		return db.Init(gormCfg)
	})

	validationJob := job.NewBankAccountValidationJob(validationSvc)
	app.AddCronJob("*/5 * * * * *", "bank_account_validation", validationJob.Run)

	registerRoutes(app, validationSvc)

	return app
}

func initPGProvider(app *gofr.App) pg_provider.BankAccountValidator {
	mode := app.Config.GetOrDefault("PG_PROVIDER_MODE", "mock")

	switch mode {
	case "live":
		baseURL := app.Config.GetOrDefault("PG_PROVIDER_LIVE_BASE_URL", "https://api.xendit.co")

		httpSvc := gofrservice.NewHTTPService(
			baseURL,
			app.Logger(),
			app.Metrics(),
			&gofrservice.CircuitBreakerConfig{
				Threshold: intFromConfig(app, "CB_THRESHOLD", "5"),
				Interval:  time.Duration(intFromConfig(app, "CB_INTERVAL_SECONDS", "30")) * time.Second,
			},
		)

		return xendit.NewClient(xendit.Config{
			APIKey:         app.Config.Get("PG_PROVIDER_LIVE_API_KEY"),
			APISecret:      app.Config.Get("PG_PROVIDER_LIVE_API_SECRET"),
			RetryMax:       intFromConfig(app, "RETRY_MAX", "2"),
			RetryBaseDelay: time.Duration(intFromConfig(app, "RETRY_BASE_DELAY_MS", "500")) * time.Millisecond,
			RetryMaxDelay:  time.Duration(intFromConfig(app, "RETRY_MAX_DELAY_MS", "5000")) * time.Millisecond,
		}, httpSvc)
	default:
		return xenditmock.NewClient(xenditmock.Config{})
	}
}

func intFromConfig(app *gofr.App, key, fallback string) int {
	value := app.Config.GetOrDefault(key, fallback)

	n, err := strconv.Atoi(value)
	if err != nil {
		n, _ = strconv.Atoi(fallback)
	}

	return n
}
