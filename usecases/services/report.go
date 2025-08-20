package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"os"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/esapi"
	"github.com/vnFuhung2903/vcs-report-service/dto"
	"github.com/vnFuhung2903/vcs-report-service/entities"
	"github.com/vnFuhung2903/vcs-report-service/interfaces"
	"github.com/vnFuhung2903/vcs-report-service/pkg/env"
	"github.com/vnFuhung2903/vcs-report-service/pkg/logger"
	"go.uber.org/zap"
	"gopkg.in/gomail.v2"
)

type IReportService interface {
	SendEmail(ctx context.Context, to string, totalCount int, onCount int, offCount int, totalUptime float64, startTime time.Time, endTime time.Time) error
	CalculateReportStatistic(statusList map[string][]dto.EsStatus, overlapStatusList map[string][]dto.EsStatus, startTime time.Time, endTime time.Time) (int, int, float64)
	GetEsStatus(ctx context.Context, limit int, startTime time.Time, endTime time.Time, order dto.SortOrder) (map[string][]dto.EsStatus, error)
}

type reportService struct {
	mailUsername string
	mailPassword string
	esClient     interfaces.IElasticsearchClient
	redisClient  interfaces.IRedisClient
	logger       logger.ILogger
}

func NewReportService(esClient interfaces.IElasticsearchClient, redisClient interfaces.IRedisClient, logger logger.ILogger, env env.GomailEnv) IReportService {
	return &reportService{
		mailUsername: env.MailUsername,
		mailPassword: env.MailPassword,
		esClient:     esClient,
		redisClient:  redisClient,
		logger:       logger,
	}
}

func (s *reportService) SendEmail(ctx context.Context, to string, totalCount int, onCount int, offCount int, totalUptime float64, startTime time.Time, endTime time.Time) error {
	emailTemplate, err := os.ReadFile("html/email.html")
	if err != nil {
		s.logger.Error("failed to read email template", zap.Error(err))
		return err
	}

	funcMap := template.FuncMap{
		"formatTime": func(t time.Time) string {
			return t.Format("2006-01-02")
		},
	}
	temp, err := template.New("report").Funcs(funcMap).Parse(string(emailTemplate))
	if err != nil {
		s.logger.Error("failed to parse template", zap.Error(err))
		return err
	}

	report := dto.ReportResponse{
		ContainerCount:    totalCount,
		ContainerOnCount:  onCount,
		ContainerOffCount: offCount,
		TotalUptime:       totalUptime,
		StartTime:         startTime,
		EndTime:           endTime,
	}

	var buf bytes.Buffer
	if err := temp.Execute(&buf, report); err != nil {
		s.logger.Error("failed to execute template", zap.Error(err))
		return err
	}

	msg := fmt.Sprintf("Container Management System Report from %s to %s", startTime.Format(time.RFC822), endTime.Format(time.RFC822))

	message := gomail.NewMessage()
	message.SetHeader("From", s.mailUsername)
	message.SetHeader("To", to)
	message.SetHeader("Subject", msg)
	message.SetBody("text/html", buf.String())

	dial := gomail.NewDialer(
		"smtp.gmail.com",
		587,
		s.mailUsername,
		s.mailPassword,
	)

	if err := dial.DialAndSend(message); err != nil {
		s.logger.Error("failed to send email", zap.Error(err))
		return err
	}

	s.logger.Info("Report sent successfully", zap.String("emailTo", to), zap.String("subject", msg))
	return nil
}

func (s *reportService) CalculateReportStatistic(statusList map[string][]dto.EsStatus, overlapStatusList map[string][]dto.EsStatus, startTime time.Time, endTime time.Time) (int, int, float64) {
	onCount := 0
	offCount := 0
	totalUptime := 0.0
	isOnline := 0

	for containerId, containerStatus := range statusList {
		previousTime := startTime
		for _, status := range containerStatus {
			if status.Status == entities.ContainerOn {
				totalUptime += min(status.LastUpdated.Sub(startTime).Hours(), float64(status.Uptime)/3600)
				isOnline = 1
			} else {
				previousTime = time.Unix(max(previousTime.Unix(), status.LastUpdated.Unix()), 0)
				isOnline = 0
			}
		}

		if len(overlapStatusList[containerId]) > 0 {
			if overlapStatusList[containerId][0].Status == entities.ContainerOn {
				onCount++
				totalUptime += min(endTime.Sub(previousTime).Hours(), float64(overlapStatusList[containerId][0].Uptime)/3600)
			} else {
				offCount++
			}
			continue
		}

		onCount += isOnline
		offCount += 1 - isOnline
	}

	return onCount, offCount, totalUptime
}

func (s *reportService) GetEsStatus(ctx context.Context, limit int, startTime time.Time, endTime time.Time, order dto.SortOrder) (map[string][]dto.EsStatus, error) {
	var body strings.Builder

	ids, err := s.redisClient.Get(ctx, "containers")
	if err != nil {
		s.logger.Error("failed to get container ids from redis", zap.Error(err))
		return nil, err
	}

	for _, id := range ids {
		meta := map[string]string{"index": "sms_container"}
		metaLine, _ := json.Marshal(meta)
		body.Write(metaLine)
		body.WriteByte('\n')

		query := map[string]interface{}{
			"query": map[string]interface{}{
				"bool": map[string]interface{}{
					"must": []interface{}{
						map[string]interface{}{"term": map[string]string{"container_id.keyword": id}},
						map[string]interface{}{
							"range": map[string]interface{}{
								"last_updated": map[string]string{
									"gte": startTime.Format(time.RFC3339),
									"lt":  endTime.Format(time.RFC3339),
								},
							},
						},
					},
				},
			},
			"size": limit,
			"sort": []interface{}{
				map[string]interface{}{"counter": map[string]string{"order": string(order)}},
			},
		}
		queryLine, _ := json.Marshal(query)
		body.Write(queryLine)
		body.WriteByte('\n')
	}

	req := esapi.MsearchRequest{
		Body: strings.NewReader(body.String()),
	}
	res, err := s.esClient.Do(ctx, req)
	if err != nil {
		s.logger.Error("failed to msearch elasticsearch status", zap.Error(err))
		return nil, err
	}
	defer res.Body.Close()

	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		s.logger.Error("failed to read response body", zap.Error(err))
		return nil, err
	}

	var parsed struct {
		Responses []struct {
			Hits struct {
				Hits []struct {
					ID     string       `json:"_id"`
					Source dto.EsStatus `json:"_source"`
				} `json:"hits"`
			} `json:"hits"`
		} `json:"responses"`
	}
	if err := json.Unmarshal(bodyBytes, &parsed); err != nil {
		s.logger.Error("failed to decode response body", zap.Error(err))
		return nil, err
	}

	results := make(map[string][]dto.EsStatus)
	for i, response := range parsed.Responses {
		containerId := ids[i]
		for _, hit := range response.Hits.Hits {
			results[containerId] = append(results[containerId], hit.Source)
		}
	}
	s.logger.Info("elasticsearch status retrieved successfully", zap.Int("containers_count", len(results)))
	return results, nil
}
