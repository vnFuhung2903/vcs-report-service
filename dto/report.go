package dto

import (
	"time"

	"github.com/vnFuhung2903/vcs-report-service/entities"
)

type ReportRequest struct {
	StartTime string `form:"start_time" binding:"required"`
	EndTime   string `form:"end_time"`
	Email     string `form:"email" binding:"required,email"`
}

type ReportResponse struct {
	ContainerCount    int       `json:"container_count"`
	ContainerOnCount  int       `json:"container_on_count"`
	ContainerOffCount int       `json:"container_off_count"`
	TotalUptime       float64   `json:"total_uptime"`
	StartTime         time.Time `json:"start_time"`
	EndTime           time.Time `json:"end_time"`
}

type EsStatus struct {
	ContainerId string                   `json:"container_id"`
	Status      entities.ContainerStatus `json:"status"`
	Uptime      int64                    `json:"uptime"`
	LastUpdated time.Time                `json:"last_updated"`
	Counter     int64                    `json:"counter"`
}

type EsStatusUpdate struct {
	ContainerId string                   `json:"container_id"`
	Status      entities.ContainerStatus `json:"status"`
}

type SortOrder string

const (
	Asc SortOrder = "asc"
	Dsc SortOrder = "desc"
)
