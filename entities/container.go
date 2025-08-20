package entities

import (
	"time"
)

type Container struct {
	ContainerId   string          `gorm:"primaryKey"`
	Status        ContainerStatus `gorm:"type:varchar(10);not null"`
	CreatedAt     time.Time       `gorm:"autoCreateTime"`
	UpdatedAt     time.Time       `gorm:"autoUpdateTime"`
	ContainerName string          `gorm:"unique;not null"`
	Ipv4          string          `gorm:"not null"`
}

type ContainerStatus string

const (
	ContainerOn  ContainerStatus = "ON"
	ContainerOff ContainerStatus = "OFF"
)
