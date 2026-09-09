package job

import (
	"gofr.dev/pkg/gofr"

	"lab-disbursement-service/internal/service"
)

type BankAccountValidationJob struct {
	svc *service.BankAccountValidationService
}

func NewBankAccountValidationJob(svc *service.BankAccountValidationService) *BankAccountValidationJob {
	return &BankAccountValidationJob{svc: svc}
}

func (j *BankAccountValidationJob) Run(ctx *gofr.Context) {
	if err := j.svc.ProcessPendingValidations(ctx); err != nil {
		ctx.Logger.Errorf("bank_account_validation: job failed: %v", err)
		return
	}

	ctx.Logger.Info("bank_account_validation: job completed")
}
