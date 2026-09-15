package job

import (
	"gofr.dev/pkg/gofr"

	"lab-disbursement-service/internal/service"
)

type DisbursementJob struct {
	svc *service.DisbursementService
}

func NewDisbursementJob(svc *service.DisbursementService) *DisbursementJob {
	return &DisbursementJob{svc: svc}
}

func (j *DisbursementJob) Run(ctx *gofr.Context) {
	if err := j.svc.ProcessPendingDisbursements(ctx); err != nil {
		ctx.Logger.Errorf("disbursement: job failed: %v", err)
		return
	}

	ctx.Logger.Info("disbursement: job completed")
}
