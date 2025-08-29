package workers

import (
	"context"
	"sync"
	"time"

	"github.com/vnFuhung2903/vcs-report-service/dto"
	"github.com/vnFuhung2903/vcs-report-service/pkg/logger"
	"github.com/vnFuhung2903/vcs-report-service/usecases/services"
	"go.uber.org/zap"
)

type IReportkWorker interface {
	Start(numWorkers int)
	Stop()
}

type reportkWorker struct {
	reportService services.IReportService
	email         string
	logger        logger.ILogger
	interval      time.Duration
	ctx           context.Context
	cancel        context.CancelFunc
	wg            *sync.WaitGroup
}

func NewReportkWorker(
	reportService services.IReportService,
	email string,
	logger logger.ILogger,
	interval time.Duration,
) IReportkWorker {
	ctx, cancel := context.WithCancel(context.Background())
	return &reportkWorker{
		reportService: reportService,
		email:         email,
		logger:        logger,
		interval:      interval,
		ctx:           ctx,
		cancel:        cancel,
		wg:            &sync.WaitGroup{},
	}
}

func (w *reportkWorker) Start(numWorkers int) {
	w.wg.Add(numWorkers)
	go w.run()
}

func (w *reportkWorker) Stop() {
	w.cancel()
	w.wg.Wait()
}

func (w *reportkWorker) run() {
	defer w.wg.Done()

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-w.ctx.Done():
			w.logger.Info("daily report workers stopped")
			return
		case <-ticker.C:
			w.report()
		}
	}
}

func (w *reportkWorker) report() {
	endTime := time.Now()
	startTime := endTime.Add(-w.interval)

	statusList, err := w.reportService.GetEsStatus(w.ctx, 10000, startTime, endTime, dto.Asc)
	if err != nil {
		w.logger.Error("failed to retrieve elasticsearch status", zap.Error(err))
		return
	}

	overlapStatusList, err := w.reportService.GetEsStatus(w.ctx, 1, endTime, time.Now(), dto.Asc)
	if err != nil {
		w.logger.Error("failed to retrieve elasticsearch status", zap.Error(err))
		return
	}

	onCount, offCount, totalUptime := w.reportService.CalculateReportStatistic(statusList, overlapStatusList, startTime, endTime)

	if err := w.reportService.SendEmail(w.ctx, w.email, onCount+offCount, onCount, offCount, totalUptime, startTime, endTime); err != nil {
		w.logger.Error("failed to email daily report", zap.Error(err))
		return
	}
	w.logger.Info("daily report emailed successfully")
}
