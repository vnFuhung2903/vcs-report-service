package workers

import (
	"errors"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"
	"github.com/vnFuhung2903/vcs-report-service/dto"
	"github.com/vnFuhung2903/vcs-report-service/entities"
	"github.com/vnFuhung2903/vcs-report-service/mocks/logger"
	"github.com/vnFuhung2903/vcs-report-service/mocks/middlewares"
	"github.com/vnFuhung2903/vcs-report-service/mocks/services"
)

type ReportHandlerSuite struct {
	suite.Suite
	ctrl              *gomock.Controller
	reportWorker      IReportkWorker
	mockReportService *services.MockIReportService
	mockJWTMiddleware *middlewares.MockIJWTMiddleware
	mockLogger        *logger.MockILogger
}

func (s *ReportHandlerSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.mockReportService = services.NewMockIReportService(s.ctrl)
	s.mockJWTMiddleware = middlewares.NewMockIJWTMiddleware(s.ctrl)
	s.mockLogger = logger.NewMockILogger(s.ctrl)

	s.mockJWTMiddleware.EXPECT().
		RequireScope("report:mail").
		Return(func(c *gin.Context) {
			c.Next()
		}).
		AnyTimes()

	s.reportWorker = NewReportkWorker(s.mockReportService, "test@example.com", s.mockLogger, 2*time.Second)
}

func (s *ReportHandlerSuite) TearDownTest() {
	s.ctrl.Finish()
}

func TestReportHandlerSuite(t *testing.T) {
	suite.Run(t, new(ReportHandlerSuite))
}

func (s *ReportHandlerSuite) TestSendEmail() {
	baseTime := time.Now()

	statusList := map[string][]dto.EsStatus{
		"container1": {
			{ContainerId: "container1", Status: entities.ContainerOn, Uptime: int64(3600), LastUpdated: baseTime.Add(-210 * time.Minute)},
			{ContainerId: "container1", Status: entities.ContainerOff, Uptime: int64(1800), LastUpdated: baseTime.Add(-3 * time.Hour)},
			{ContainerId: "container1", Status: entities.ContainerOn, Uptime: int64(3600), LastUpdated: baseTime.Add(-2 * time.Hour)},
		},
		"container2": {
			{ContainerId: "container2", Status: entities.ContainerOff, Uptime: int64(7200), LastUpdated: baseTime.Add(-1 * time.Minute)},
		},
	}

	overlapStatusList := map[string][]dto.EsStatus{
		"container1": {},
		"container2": {},
	}

	s.mockReportService.EXPECT().
		GetEsStatus(gomock.Any(), 10000, gomock.Any(), gomock.Any(), dto.Asc).
		Return(statusList, nil)

	s.mockReportService.EXPECT().
		GetEsStatus(gomock.Any(), 1, gomock.Any(), gomock.Any(), dto.Asc).
		Return(overlapStatusList, nil)

	s.mockReportService.EXPECT().
		CalculateReportStatistic(statusList, overlapStatusList, gomock.Any(), gomock.Any()).
		Return(1, 1, 50.0)

	s.mockReportService.EXPECT().
		SendEmail(gomock.Any(), "test@example.com", 2, 1, 1, 50.0, gomock.Any(), gomock.Any()).
		Return(nil)

	s.mockLogger.EXPECT().Info("daily report sent successfully", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	s.mockLogger.EXPECT().Info("daily report workers stopped").AnyTimes()

	s.reportWorker.Start()
	time.Sleep(3 * time.Second)

	s.reportWorker.Stop()
}

func (s *ReportHandlerSuite) TestSendEmailGetEsStatusError() {
	s.mockReportService.EXPECT().
		GetEsStatus(gomock.Any(), 10000, gomock.Any(), gomock.Any(), dto.Asc).
		Return(map[string][]dto.EsStatus{}, errors.New("elasticsearch error"))

	s.mockLogger.EXPECT().Error("failed to retrieve elasticsearch status", gomock.Any()).AnyTimes()
	s.mockLogger.EXPECT().Info("daily report workers stopped").AnyTimes()

	s.reportWorker.Start()
	time.Sleep(3 * time.Second)

	s.reportWorker.Stop()
}

func (s *ReportHandlerSuite) TestSendEmailGetEsStatusOverlapError() {
	baseTime := time.Now()
	statusList := map[string][]dto.EsStatus{
		"container1": {
			{ContainerId: "container1", Status: entities.ContainerOn, Uptime: int64(3600), LastUpdated: baseTime.Add(-210 * time.Minute)},
			{ContainerId: "container1", Status: entities.ContainerOff, Uptime: int64(1800), LastUpdated: baseTime.Add(-3 * time.Hour)},
			{ContainerId: "container1", Status: entities.ContainerOn, Uptime: int64(3600), LastUpdated: baseTime.Add(-2 * time.Hour)},
		},
	}

	s.mockReportService.EXPECT().
		GetEsStatus(gomock.Any(), 10000, gomock.Any(), gomock.Any(), dto.Asc).
		Return(statusList, nil)

	s.mockReportService.EXPECT().
		GetEsStatus(gomock.Any(), 1, gomock.Any(), gomock.Any(), dto.Asc).
		Return(map[string][]dto.EsStatus{}, errors.New("elasticsearch error"))

	s.mockLogger.EXPECT().Error("failed to retrieve elasticsearch status", gomock.Any()).AnyTimes()
	s.mockLogger.EXPECT().Info("daily report workers stopped").AnyTimes()

	s.reportWorker.Start()
	time.Sleep(3 * time.Second)

	s.reportWorker.Stop()
}

func (s *ReportHandlerSuite) TestSendEmailSendEmailServiceError() {
	baseTime := time.Now()
	statusList := map[string][]dto.EsStatus{
		"container1": {
			{ContainerId: "container1", Status: entities.ContainerOn, Uptime: int64(3600), LastUpdated: baseTime.Add(-210 * time.Minute)},
			{ContainerId: "container1", Status: entities.ContainerOff, Uptime: int64(1800), LastUpdated: baseTime.Add(-3 * time.Hour)},
			{ContainerId: "container1", Status: entities.ContainerOn, Uptime: int64(3600), LastUpdated: baseTime.Add(-2 * time.Hour)},
		},
	}

	overlapStatusList := map[string][]dto.EsStatus{
		"container1": {},
	}

	s.mockReportService.EXPECT().
		GetEsStatus(gomock.Any(), 10000, gomock.Any(), gomock.Any(), dto.Asc).
		Return(statusList, nil)

	s.mockReportService.EXPECT().
		GetEsStatus(gomock.Any(), 1, gomock.Any(), gomock.Any(), dto.Asc).
		Return(overlapStatusList, nil)

	s.mockReportService.EXPECT().
		CalculateReportStatistic(statusList, overlapStatusList, gomock.Any(), gomock.Any()).
		Return(1, 0, 100.0)

	s.mockReportService.EXPECT().
		SendEmail(gomock.Any(), "test@example.com", 1, 1, 0, 100.0, gomock.Any(), gomock.Any()).
		Return(errors.New("service error"))

	s.mockLogger.EXPECT().Error("failed to email daily report", gomock.Any()).AnyTimes()
	s.mockLogger.EXPECT().Info("daily report workers stopped").AnyTimes()

	s.reportWorker.Start()
	time.Sleep(3 * time.Second)

	s.reportWorker.Stop()
}
