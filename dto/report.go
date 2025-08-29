package dto

import (
	"time"
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
