package services

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/elastic/go-elasticsearch/esapi"
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

type MockElasticsearchResponse struct {
	Body       io.ReadCloser
	StatusCode int
}

func NewMockElasticsearchResponse(body string, statusCode int) *esapi.Response {
	return &esapi.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
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

func (s *ReportServiceSuite) TestGetEsStatus() {
	ctx := context.Background()
	startTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endTime := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
	limit := 1000

	containers := []entities.ContainerWithStatus{
		{ContainerId: "container1", Status: entities.ContainerOn},
		{ContainerId: "container2", Status: entities.ContainerOff},
	}

	s.redisClient.EXPECT().
		Get(ctx, "containers").
		Return(containers, nil)

	esResponse := `{
        "responses": [
            {
                "hits": {
                    "hits": [
                        {
                            "_id": "1",
                            "_source": {
                                "container_id": "container1",
                                "status": "ON",
                                "uptime": 3600,
                                "last_updated": "2024-01-01T12:00:00Z",
                                "counter": 1
                            }
                        }
                    ]
                }
            },
            {
                "hits": {
                    "hits": [
                        {
                            "_id": "2",
                            "_source": {
                                "container_id": "container2",
                                "status": "OFF",
                                "uptime": 1800,
                                "last_updated": "2024-01-01T13:00:00Z",
                                "counter": 2
                            }
                        }
                    ]
                }
            }
        ]
    }`

	mockResponse := NewMockElasticsearchResponse(esResponse, 200)

	s.esClient.EXPECT().
		Do(ctx, gomock.Any()).
		Return(mockResponse, nil)

	s.logger.EXPECT().
		Info("elasticsearch status retrieved successfully", gomock.Any()).
		Times(1)

	result, err := s.reportService.GetEsStatus(ctx, limit, startTime, endTime, dto.Asc)

	s.NoError(err)
	s.Len(result, 2)
	s.Contains(result, "container1")
	s.Contains(result, "container2")
	s.Len(result["container1"], 1)
	s.Len(result["container2"], 1)
	s.Equal("container1", result["container1"][0].ContainerId)
	s.Equal(entities.ContainerOn, result["container1"][0].Status)
	s.Equal("container2", result["container2"][0].ContainerId)
	s.Equal(entities.ContainerOff, result["container2"][0].Status)
}

func (s *ReportServiceSuite) TestGetEsStatusRedisError() {
	ctx := context.Background()
	startTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endTime := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
	limit := 1000

	expectedError := errors.New("redis connection failed")

	s.redisClient.EXPECT().
		Get(ctx, "containers").
		Return(nil, expectedError)

	s.logger.EXPECT().
		Error("failed to get container ids from redis", gomock.Any()).
		Times(1)

	result, err := s.reportService.GetEsStatus(ctx, limit, startTime, endTime, dto.Asc)

	s.Error(err)
	s.Nil(result)
	s.Equal(expectedError, err)
}

func (s *ReportServiceSuite) TestGetEsStatusElasticsearchError() {
	ctx := context.Background()
	startTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endTime := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
	limit := 1000

	containers := []entities.ContainerWithStatus{
		{ContainerId: "container1", Status: entities.ContainerOn},
	}

	expectedError := errors.New("elasticsearch connection failed")

	s.redisClient.EXPECT().
		Get(ctx, "containers").
		Return(containers, nil)

	s.esClient.EXPECT().
		Do(ctx, gomock.Any()).
		Return(nil, expectedError)

	s.logger.EXPECT().
		Error("failed to msearch elasticsearch status", gomock.Any()).
		Times(1)

	result, err := s.reportService.GetEsStatus(ctx, limit, startTime, endTime, dto.Asc)

	s.Error(err)
	s.Nil(result)
	s.Equal(expectedError, err)
}

func (s *ReportServiceSuite) TestGetEsStatusInvalidJSONResponse() {
	ctx := context.Background()
	startTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endTime := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
	limit := 1000

	containers := []entities.ContainerWithStatus{
		{ContainerId: "container1", Status: entities.ContainerOn},
	}

	s.redisClient.EXPECT().
		Get(ctx, "containers").
		Return(containers, nil)

	invalidJSON := `{"invalid": json}`
	mockResponse := NewMockElasticsearchResponse(invalidJSON, 200)

	s.esClient.EXPECT().
		Do(ctx, gomock.Any()).
		Return(mockResponse, nil)

	s.logger.EXPECT().
		Error("failed to decode response body", gomock.Any()).
		Times(1)

	result, err := s.reportService.GetEsStatus(ctx, limit, startTime, endTime, dto.Asc)

	s.Error(err)
	s.Nil(result)
}
