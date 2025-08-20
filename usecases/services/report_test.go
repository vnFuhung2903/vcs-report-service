package services

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"

	"github.com/vnFuhung2903/vcs-report-service/dto"
	"github.com/vnFuhung2903/vcs-report-service/entities"
	"github.com/vnFuhung2903/vcs-report-service/mocks/interfaces"
	"github.com/vnFuhung2903/vcs-report-service/mocks/logger"
	"github.com/vnFuhung2903/vcs-report-service/pkg/env"
)

type ReportServiceSuite struct {
	suite.Suite
	ctrl          *gomock.Controller
	esClient      *interfaces.MockIElasticsearchClient
	redisClient   *interfaces.MockIRedisClient
	reportService IReportService
	logger        *logger.MockILogger
	ctx           context.Context
	sampleReport  *dto.ReportResponse
}

func (s *ReportServiceSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.esClient = interfaces.NewMockIElasticsearchClient(s.ctrl)
	s.redisClient = interfaces.NewMockIRedisClient(s.ctrl)
	s.logger = logger.NewMockILogger(s.ctrl)

	s.reportService = NewReportService(s.esClient, s.redisClient, s.logger, env.GomailEnv{
		MailUsername: "test@gmail.com",
		MailPassword: "testpass",
	})
	s.ctx = context.Background()

	s.sampleReport = &dto.ReportResponse{
		ContainerCount:    10,
		ContainerOnCount:  7,
		ContainerOffCount: 3,
		TotalUptime:       24.5,
		StartTime:         time.Now().Add(-24 * time.Hour),
		EndTime:           time.Now(),
	}

	err := os.MkdirAll("html", 0755)
	if err != nil {
		s.T().Fatal("Failed to create html directory:", err)
	}

	htmlContent := `<!DOCTYPE html>
<html>
<head><title>Test Report</title></head>
<body>
	<h1>Daily Container Report</h1>
	<p>{{ .StartTime | formatTime }} - {{ .EndTime | formatTime }}</p>
	<p>Total Container: {{ .ContainerCount }}</p>
	<p>Online Containers: {{ .ContainerOnCount }}</p>
	<p>Offline Containers: {{ .ContainerOffCount }}</p>
	<p>Total Uptime: {{ .TotalUptime }}h</p>
</body>
</html>`

	err = os.WriteFile("html/email.html", []byte(htmlContent), 0644)
	if err != nil {
		s.T().Fatal("Failed to create html file:", err)
	}
}

func (s *ReportServiceSuite) TearDownTest() {
	os.RemoveAll("html")
	s.ctrl.Finish()
}

func TestReportServiceSuite(t *testing.T) {
	suite.Run(t, new(ReportServiceSuite))
}

// func (s *ReportServiceSuite) TestSendEmail() {
// 	s.reportService = NewReportService(s.logger, env.GomailEnv{
// 		MailUsername: "hung29032004@gmail.com",
// 		MailPassword: "",
// 	})
// 	s.logger.EXPECT().Info("Report sent successfully", gomock.Any(), gomock.Any()).Times(1)
// 	err := s.reportService.SendEmail(s.ctx, "hung29032004@gmail.com", 10, 7, 3, 24.5, s.sampleReport.StartTime, s.sampleReport.EndTime)
// 	s.Nil(err)
// }

func (s *ReportServiceSuite) TestSendEmailError() {
	s.logger.EXPECT().Error("failed to send email", gomock.Any()).Times(1)
	err := s.reportService.SendEmail(s.ctx, "recipient@example.com", 10, 7, 3, 24.5, s.sampleReport.StartTime, s.sampleReport.EndTime)
	s.Error(err)
}

func (s *ReportServiceSuite) TestSendEmailTemplateNotFound() {
	os.Remove("html/email.html")
	s.logger.EXPECT().Error("failed to read email template", gomock.Any()).Times(1)
	err := s.reportService.SendEmail(s.ctx, "recipient@example.com", 10, 7, 3, 24.5, s.sampleReport.StartTime, s.sampleReport.EndTime)
	s.Error(err)
}

func (s *ReportServiceSuite) TestSendEmailInvalidTemplate() {
	invalidTemplate := `{{invalid template syntax`
	err := os.WriteFile("html/email.html", []byte(invalidTemplate), 0644)
	s.NoError(err)

	s.logger.EXPECT().Error("failed to parse template", gomock.Any()).Times(1)
	err = s.reportService.SendEmail(s.ctx, "recipient@example.com", 10, 7, 3, 24.5, s.sampleReport.StartTime, s.sampleReport.EndTime)
	s.Error(err)
}

func (s *ReportServiceSuite) TestSendEmailTemplateExecutionError() {
	invalidTemplate := `<html><body>{{.NonExistentField}}</body></html>`
	err := os.WriteFile("html/email.html", []byte(invalidTemplate), 0644)
	s.NoError(err)

	s.logger.EXPECT().Error("failed to execute template", gomock.Any()).Times(1)
	err = s.reportService.SendEmail(s.ctx, "recipient@example.com", 10, 7, 3, 24.5, s.sampleReport.StartTime, s.sampleReport.EndTime)
	s.Error(err)
}

func (s *ReportServiceSuite) TestCalculateReportStatistic() {
	baseTime := time.Now()
	endTime := baseTime
	startTime := endTime.Add(-4 * time.Hour)
	statusList := map[string][]dto.EsStatus{
		"container1": {
			{ContainerId: "container1", Status: entities.ContainerOn, Uptime: int64(3600), LastUpdated: baseTime.Add(-210 * time.Minute)},
			{ContainerId: "container1", Status: entities.ContainerOff, Uptime: int64(1800), LastUpdated: baseTime.Add(-3 * time.Hour)},
			{ContainerId: "container1", Status: entities.ContainerOn, Uptime: int64(3600), LastUpdated: baseTime.Add(-2 * time.Hour)},
		},
		"container2": {
			{ContainerId: "container2", Status: entities.ContainerOff, Uptime: int64(7200), LastUpdated: baseTime.Add(-1 * time.Minute)},
		},
		"container3": {},
	}

	overlapStatusList := map[string][]dto.EsStatus{
		"container1": {
			{ContainerId: "container1", Status: entities.ContainerOff, Uptime: int64(7200), LastUpdated: baseTime},
		},
		"container2": {},
		"container3": {
			{ContainerId: "container3", Status: entities.ContainerOn, Uptime: int64(1800), LastUpdated: baseTime},
		},
	}

	onCount, offCount, totalUptime := s.reportService.CalculateReportStatistic(statusList, overlapStatusList, startTime, endTime)

	s.Equal(1, onCount)
	s.Equal(2, offCount)
	s.Equal(float64(2), totalUptime)
}
