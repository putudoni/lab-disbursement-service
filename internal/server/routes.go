package server

import (
	"strconv"

	"gofr.dev/pkg/gofr"

	"lab-disbursement-service/internal/model"
	"lab-disbursement-service/internal/service"
)

func registerRoutes(app *gofr.App, validationSvc *service.BankAccountValidationService, disbursementSvc *service.DisbursementService) {
	app.GET("/", func(c *gofr.Context) (interface{}, error) {
		return map[string]string{"service": "lab-disbursement-service", "status": "ok"}, nil
	})

	app.GET("/bank-accounts/{id}", getBankAccountHandler(validationSvc))
	app.GET("/disbursements/{id}", getDisbursementHandler(disbursementSvc))
}

func getDisbursementHandler(svc *service.DisbursementService) func(*gofr.Context) (interface{}, error) {
	return func(c *gofr.Context) (interface{}, error) {
		id, err := strconv.ParseInt(c.PathParam("id"), 10, 64)
		if err != nil {
			return nil, model.ErrInvalidDisbursementID
		}

		disbursement, err := svc.GetDisbursement(c, id)
		if err != nil {
			return nil, err
		}
		if disbursement == nil {
			return nil, model.ErrDisbursementNotFound
		}

		svc.RecordView(c, *disbursement, "api:"+c.HostName(), c.GetCorrelationID())

		return disbursement, nil
	}
}

func getBankAccountHandler(svc *service.BankAccountValidationService) func(*gofr.Context) (interface{}, error) {
	return func(c *gofr.Context) (interface{}, error) {
		id, err := strconv.ParseInt(c.PathParam("id"), 10, 64)
		if err != nil {
			return nil, model.ErrInvalidBankAccountID
		}

		account, err := svc.GetBankAccount(c, id)
		if err != nil {
			return nil, err
		}
		if account == nil {
			return nil, model.ErrBankAccountNotFound
		}

		svc.RecordView(c, *account, "api:"+c.HostName(), c.GetCorrelationID())

		return account, nil
	}
}
